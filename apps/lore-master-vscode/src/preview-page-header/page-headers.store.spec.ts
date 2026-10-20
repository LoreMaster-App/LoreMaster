import { join } from 'node:path'
import { PageHeaders, STALE_AFTER_MS } from './page-headers.store'
import type { HeaderEngine } from './page-header.contract'

const ROOT = join('work', 'repo')
const FILE = join(ROOT, 'docs', 'guide.md')

function engineWith (status: string): HeaderEngine & { calls: string[] } {
  const calls: string[] = []

  return {
    calls,
    request: (method: string) => {
      calls.push(method)
      if (method === 'settings/read') {
        return Promise.resolve({ settings: { outputs: [{}] }, displayNames: ['Confluence · DOCS'] }) as never
      }

      return Promise.resolve({ nodes: [{ path: 'docs/guide.md', pageTitle: 'Team: Guide', status }] }) as never
    },
  }
}

const settle = (): Promise<void> => new Promise(resolve => { setTimeout(resolve, 0) })

describe('PageHeaders', () => {
  it('knows nothing of a file at first, asks the engine once, and says so when the answer is in', async () => {
    const engine = engineWith('synced')
    const changed = jest.fn()
    const headers = new PageHeaders(engine, () => ROOT, changed, () => 0)

    expect(headers.headersOf(FILE)).toBeUndefined()
    expect(headers.headersOf(FILE)).toBeUndefined()
    await settle()

    expect(engine.calls).toEqual(['settings/read', 'workspace/tree'])
    expect(changed).toHaveBeenCalledTimes(1)
    expect(headers.headersOf(FILE)).toEqual([{ storage: 'Confluence · DOCS', pageTitle: 'Team: Guide', status: 'synced' }])
  })

  it('has no header for a file outside every workspace folder', () => {
    const engine = engineWith('synced')
    const headers = new PageHeaders(engine, () => undefined, jest.fn(), () => 0)

    expect(headers.headersOf(FILE)).toBeUndefined()
    expect(engine.calls).toEqual([])
  })

  it('asks again once the answer is stale and only says so when it changed', async () => {
    let clock = 0
    const changed = jest.fn()
    const first = engineWith('synced')
    const headers = new PageHeaders(first, () => ROOT, changed, () => clock)
    headers.headersOf(FILE)
    await settle()

    clock += STALE_AFTER_MS - 1
    headers.headersOf(FILE)
    await settle()
    expect(first.calls).toHaveLength(2)

    clock += 2
    headers.headersOf(FILE)
    await settle()
    expect(first.calls).toHaveLength(4)
    expect(changed).toHaveBeenCalledTimes(1)
  })

  it('keeps what it knew when the engine fails', async () => {
    const failing: HeaderEngine = { request: () => Promise.reject(new Error('not ready')) }
    const changed = jest.fn()
    const headers = new PageHeaders(failing, () => ROOT, changed, () => 0)

    headers.headersOf(FILE)
    await settle()

    expect(changed).not.toHaveBeenCalled()
    expect(headers.headersOf(FILE)).toBeUndefined()
  })
})
