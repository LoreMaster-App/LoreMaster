import { execFileSync } from 'node:child_process'
import { existsSync, mkdtempSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { createEngineClient, type EngineClient } from './engine-rpc.client'
import { PING_METHOD, type PingResult } from './rpc-protocol.contract'

// Spawns the REAL Go engine and pings it over stdio, proving the client's framing
// matches the server's. Gated: it runs when LORE_MASTER_ENGINE_BIN points at a built
// binary, or when `go` is on PATH to build one (CI). Without either it is skipped, so a
// checkout with no Go toolchain (and no display) stays green. The build happens once in
// beforeAll, not at module load, so its time counts against a generous timeout.

const repoRoot = resolve(__dirname, '../../../..')

function goAvailable (): boolean {
  try {
    execFileSync('go', ['version'], { stdio: 'pipe' })

    return true
  } catch {
    return false
  }
}

function buildEngine (): string {
  const directory = mkdtempSync(join(tmpdir(), 'lore-engine-'))
  const binary = join(directory, process.platform === 'win32' ? 'lore-master-engine.exe' : 'lore-master-engine')
  execFileSync('go', ['build', '-o', binary, './apps/lore-master-engine'], { cwd: repoRoot, stdio: 'pipe' })

  return binary
}

const configured = process.env.LORE_MASTER_ENGINE_BIN
const canRun = (configured && existsSync(configured)) || goAvailable()
const describeWithEngine = canRun ? describe : describe.skip

describeWithEngine('engine-rpc against the real engine', () => {
  let binaryPath: string
  let client: EngineClient

  beforeAll(() => {
    binaryPath = configured && existsSync(configured) ? configured : buildEngine()
  }, 180_000)

  afterAll(() => client?.dispose())

  it('spawns the engine and answers ping', async () => {
    client = createEngineClient({ binaryPath })

    const result = await client.request<PingResult>(PING_METHOD)

    expect(result.pong).toBe('pong')
    expect(typeof result.version).toBe('string')
  }, 20_000)
})
