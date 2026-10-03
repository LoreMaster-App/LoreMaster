import { spawn as nodeSpawn } from 'node:child_process'
import { createMessageConnection, type MessageConnection, StreamMessageReader, StreamMessageWriter } from 'vscode-jsonrpc/node'

/**
 * The child process the client drives. A structural subset of Node's
 * `ChildProcess`, so tests can stand in a fake over in-memory streams without a real
 * process (and the real {@link spawnEngine} satisfies it).
 */
export interface EngineChild {
  stdin:  NodeJS.WritableStream
  stdout: NodeJS.ReadableStream
  stderr: NodeJS.ReadableStream
  on (event: 'exit', listener: (code: number | null) => void): void
  on (event: 'error', listener: (error: Error) => void): void
  kill (): void
}

/** Starts the engine binary. Injected so tests can supply a fake child. */
export type SpawnEngine = (binaryPath: string) => EngineChild

/** How to build {@link createEngineClient}. */
export interface EngineClientOptions {
  /** Absolute path to the engine binary (see `resolveEngineBinary`). */
  binaryPath: string
  /** Defaults to spawning the real binary with piped stdio. */
  spawn?:     SpawnEngine
  /** Receives the engine's stderr lines (it logs there). */
  onLog?:     (line: string) => void
  /** Waits between restart attempts after a crash; the last value repeats. The first
   *  start never waits. Defaults to 250ms, 1s, 5s. */
  backoffMs?: readonly number[]
  /** Sleeps for `ms`; injected so tests need no real timers. */
  delay?:     (ms: number) => Promise<void>
}

/** A lazily-started connection to the engine over JSON-RPC. */
export interface EngineClient {
  /** Calls one engine method, starting (or restarting) the engine if needed. */
  request<R>(method: string, params?: unknown): Promise<R>
  /** Kills the engine and closes the connection. Further requests reject. */
  dispose (): void
}

const DEFAULT_BACKOFF_MS = [250, 1000, 5000] as const

const spawnEngine: SpawnEngine = binaryPath => nodeSpawn(binaryPath, [], { stdio: ['pipe', 'pipe', 'pipe'] })

const sleep = (ms: number): Promise<void> => new Promise(resolve => { setTimeout(resolve, ms) })

/**
 * Creates a client that spawns the engine on first use and speaks JSON-RPC 2.0 over its
 * stdio (Content-Length framing, which the Go server uses too).
 *
 * The engine is started lazily and kept for reuse. If it crashes, the connection is torn
 * down and the next request restarts it after a backoff that grows with consecutive
 * crashes and resets on the first request that succeeds. `dispose` kills it for good.
 */
export function createEngineClient (options: EngineClientOptions): EngineClient {
  const spawn = options.spawn ?? spawnEngine
  const backoff = options.backoffMs ?? DEFAULT_BACKOFF_MS
  const delay = options.delay ?? sleep

  let connection: MessageConnection | undefined
  let child: EngineChild | undefined
  let starting: Promise<MessageConnection> | undefined
  let crashes = 0
  let disposed = false

  function teardown (victim: MessageConnection): void {
    if (connection !== victim) {
      return
    }
    connection = undefined
    child = undefined
    starting = undefined
    crashes += 1
    victim.dispose()
  }

  async function launch (): Promise<MessageConnection> {
    if (crashes > 0) {
      await delay(backoff[Math.min(crashes - 1, backoff.length - 1)])
    }
    if (disposed) {
      throw new Error('engine client disposed')
    }

    const started = spawn(options.binaryPath)
    const established = createMessageConnection(new StreamMessageReader(started.stdout), new StreamMessageWriter(started.stdin))

    started.stderr.on('data', (chunk: Buffer | string) => options.onLog?.(chunk.toString().replace(/\n$/, '')))
    started.on('error', error => options.onLog?.(`engine failed to start: ${error.message}`))
    started.on('exit', () => teardown(established))
    established.onClose(() => teardown(established))
    established.onError(([error]) => options.onLog?.(`engine connection error: ${error.message}`))
    established.listen()

    connection = established
    child = started
    starting = undefined

    return established
  }

  function start (): Promise<MessageConnection> {
    if (disposed) {
      return Promise.reject(new Error('engine client disposed'))
    }
    if (connection) {
      return Promise.resolve(connection)
    }
    starting ??= launch()

    return starting
  }

  return {
    async request<R> (method: string, params?: unknown): Promise<R> {
      const established = await start()
      const result = params === undefined
        ? await established.sendRequest<R>(method)
        : await established.sendRequest<R>(method, params)
      crashes = 0

      return result
    },
    dispose (): void {
      disposed = true
      connection?.dispose()
      child?.kill()
      connection = undefined
      child = undefined
      starting = undefined
    },
  }
}
