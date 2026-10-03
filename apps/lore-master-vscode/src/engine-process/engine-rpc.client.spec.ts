import { PassThrough } from 'node:stream'
import { createMessageConnection, StreamMessageReader, StreamMessageWriter } from 'vscode-jsonrpc/node'
import { createEngineClient, type EngineChild } from './engine-rpc.client'
import { PING_METHOD, type PingResult } from '../engine-protocol'

type ExitListener = (code: number | null) => void

/** A fake engine over in-memory streams: real JSON-RPC framing, no child process. */
interface FakeEngine {
  child:     EngineChild
  stderr:    PassThrough
  crash:     (code?: number) => void
  wasKilled: () => boolean
}

function fakeEngine (ping: () => PingResult = () => ({ pong: 'pong', version: 'test' })): FakeEngine {
  const toEngine = new PassThrough()
  const fromEngine = new PassThrough()
  const stderr = new PassThrough()
  const exitListeners: ExitListener[] = []
  let killed = false

  const engine = createMessageConnection(new StreamMessageReader(toEngine), new StreamMessageWriter(fromEngine))
  engine.onRequest(PING_METHOD, ping)
  engine.listen()

  function exit (code: number | null): void {
    engine.dispose()
    for (const listener of exitListeners) {
      listener(code)
    }
  }

  const child = {
    stdin:  toEngine,
    stdout: fromEngine,
    stderr,
    on (event: string, listener: ExitListener): void {
      if (event === 'exit') {
        exitListeners.push(listener)
      }
    },
    kill (): void {
      killed = true
      exit(null)
    },
  } as unknown as EngineChild

  return {
    child,
    stderr,
    crash:     (code = 1) => { exit(code) },
    wasKilled: () => killed,
  }
}

const flush = (): Promise<void> => new Promise(resolve => { setImmediate(resolve) })

describe('createEngineClient', () => {
  it('starts the engine on the first request and returns its reply', async () => {
    const spawn = jest.fn(() => fakeEngine().child)
    const client = createEngineClient({ binaryPath: '/engine', spawn })

    const result = await client.request<PingResult>(PING_METHOD)

    expect(result).toEqual({ pong: 'pong', version: 'test' })
    expect(spawn).toHaveBeenCalledTimes(1)
    expect(spawn).toHaveBeenCalledWith('/engine')
    client.dispose()
  })

  it('reuses one engine across requests', async () => {
    const spawn = jest.fn(() => fakeEngine().child)
    const client = createEngineClient({ binaryPath: '/engine', spawn })

    await client.request(PING_METHOD)
    await client.request(PING_METHOD)

    expect(spawn).toHaveBeenCalledTimes(1)
    client.dispose()
  })

  it('passes params through to the engine', async () => {
    const seen: unknown[] = []
    const toEngine = new PassThrough()
    const fromEngine = new PassThrough()
    const engine = createMessageConnection(new StreamMessageReader(toEngine), new StreamMessageWriter(fromEngine))
    engine.onRequest('echo', (params: unknown) => {
      seen.push(params)

      return params
    })
    engine.listen()
    const child = { stdin: toEngine, stdout: fromEngine, stderr: new PassThrough(), on () {}, kill () {} } as unknown as EngineChild
    const client = createEngineClient({ binaryPath: '/engine', spawn: () => child })

    const echoed = await client.request('echo', { roots: ['.'] })

    expect(echoed).toEqual({ roots: ['.'] })
    expect(seen).toEqual([{ roots: ['.'] }])
    client.dispose()
  })

  it('restarts after a crash, waiting the backoff first', async () => {
    const engines: FakeEngine[] = []
    const spawn = jest.fn(() => {
      const engine = fakeEngine(); engines.push(engine)

      return engine.child
    })
    const delay = jest.fn(async () => {})
    const client = createEngineClient({ binaryPath: '/engine', spawn, delay, backoffMs: [250, 1000] })

    await client.request(PING_METHOD)
    engines[0].crash()
    await flush()
    const afterCrash = await client.request<PingResult>(PING_METHOD)

    expect(afterCrash.pong).toBe('pong')
    expect(spawn).toHaveBeenCalledTimes(2)
    expect(delay).toHaveBeenCalledWith(250)
    client.dispose()
  })

  it('does not wait before the very first start', async () => {
    const delay = jest.fn(async () => {})
    const client = createEngineClient({ binaryPath: '/engine', spawn: () => fakeEngine().child, delay })

    await client.request(PING_METHOD)

    expect(delay).not.toHaveBeenCalled()
    client.dispose()
  })

  it('resets the backoff after a request succeeds', async () => {
    const engines: FakeEngine[] = []
    const spawn = jest.fn(() => {
      const engine = fakeEngine(); engines.push(engine)

      return engine.child
    })
    const delay = jest.fn(async () => {})
    const client = createEngineClient({ binaryPath: '/engine', spawn, delay, backoffMs: [250, 1000] })

    await client.request(PING_METHOD)
    engines[0].crash()
    await flush()
    await client.request(PING_METHOD) // waits 250, then succeeds -> resets
    engines[1].crash()
    await flush()
    await client.request(PING_METHOD) // a first crash again -> waits 250, not 1000

    expect(delay.mock.calls).toEqual([[250], [250]])
    client.dispose()
  })

  it('kills the engine on dispose and rejects later requests', async () => {
    const engine = fakeEngine()
    const client = createEngineClient({ binaryPath: '/engine', spawn: () => engine.child })

    await client.request(PING_METHOD)
    client.dispose()

    expect(engine.wasKilled()).toBe(true)
    await expect(client.request(PING_METHOD)).rejects.toThrow('disposed')
  })

  it('forwards the engine stderr to onLog', async () => {
    const lines: string[] = []
    const engine = fakeEngine()
    const client = createEngineClient({ binaryPath: '/engine', spawn: () => engine.child, onLog: line => { lines.push(line) } })

    await client.request(PING_METHOD)
    engine.stderr.write('engine says hi\n')
    await flush()

    expect(lines).toContain('engine says hi')
    client.dispose()
  })
})
