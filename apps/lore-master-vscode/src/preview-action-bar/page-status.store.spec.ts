import { join } from 'node:path'
import { PageStatuses, STALE_AFTER_MS } from './page-status.store'
import type { StatusEngine } from './action-bar.contract'

const ROOT = join('work', 'repo')
const FILE = join(ROOT, 'docs', 'guide.md')
const OTHER = join(ROOT, 'docs', 'other.md')

function answer (method: string, status?: string): never {
  return (method === 'settings/read'
    ? Promise.resolve({ settings: { outputs: [{}] }, displayNames: ['Confluence · DOCS'] })
    : Promise.resolve({ nodes: [{ path: 'docs/guide.md', pageTitle: 'Team: Guide', status }] })) as never
}

function engineWith (status: string): StatusEngine & { calls: string[] } {
  const calls: string[] = []

  return {
    calls,
    request: (method: string) => {
      calls.push(method)

      return answer(method, status)
    },
  }
}

const settle = (): Promise<void> => new Promise(resolve => { setTimeout(resolve, 0) })

describe('PageStatuses', () => {
  it('is checking at first, asks the engine once, and says so when the answer is in', async () => {
    const engine = engineWith('synced')
    const changed = jest.fn()
    const statuses = new PageStatuses(engine, () => ROOT, changed, () => 0)

    expect(statuses.statusOf(FILE)).toEqual({ state: 'checking' })
    expect(statuses.statusOf(FILE)).toEqual({ state: 'checking' })
    await settle()

    expect(engine.calls).toEqual(['settings/read', 'workspace/tree'])
    expect(changed).toHaveBeenCalledTimes(1)
    expect(statuses.statusOf(FILE)).toEqual({ state: 'tracked', pages: [{ storage: 'Confluence · DOCS', pageTitle: 'Team: Guide', status: 'synced' }] })
    expect(statuses.statusOf(OTHER)).toEqual({ state: 'untracked' })
  })

  it('has no status for a file outside every workspace folder', () => {
    const engine = engineWith('synced')
    const statuses = new PageStatuses(engine, () => undefined, jest.fn(), () => 0)

    expect(statuses.statusOf(FILE)).toEqual({ state: 'untracked' })
    expect(engine.calls).toEqual([])
  })

  it('asks again once the answer is stale and only says so when it changed', async () => {
    let clock = 0
    const changed = jest.fn()
    const engine = engineWith('synced')
    const statuses = new PageStatuses(engine, () => ROOT, changed, () => clock)
    statuses.statusOf(FILE)
    await settle()

    clock += STALE_AFTER_MS - 1
    statuses.statusOf(FILE)
    await settle()
    expect(engine.calls).toHaveLength(2)

    clock += 2
    statuses.statusOf(FILE)
    await settle()
    expect(engine.calls).toHaveLength(4)
    expect(changed).toHaveBeenCalledTimes(1)
  })

  it('shows the reason when the engine fails, and recovers on the next ask', async () => {
    let clock = 0
    let failing = true
    const engine: StatusEngine = {
      request: (method: string) => failing ? Promise.reject(new Error('engine not ready')) : answer(method),
    }
    const changed = jest.fn()
    const statuses = new PageStatuses(engine, () => ROOT, changed, () => clock)

    statuses.statusOf(FILE)
    await settle()
    expect(statuses.statusOf(FILE)).toEqual({ state: 'unavailable', reason: 'engine not ready' })
    expect(changed).toHaveBeenCalledTimes(1)

    failing = false
    clock += STALE_AFTER_MS + 1
    statuses.statusOf(FILE)
    await settle()
    expect(statuses.statusOf(FILE).state).toBe('tracked')
    expect(changed).toHaveBeenCalledTimes(2)
  })
})
