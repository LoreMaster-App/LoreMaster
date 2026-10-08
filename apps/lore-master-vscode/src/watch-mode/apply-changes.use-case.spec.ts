import { GENERATORS_RUN_METHOD, WATCH_ROUTE_METHOD, type GeneratorRun, type WatchRouteResult } from '../engine-protocol'
import { applyChanges } from './apply-changes.use-case'

interface Scenario {
  route:      WatchRouteResult
  runs?:      GeneratorRun[]
  requests:   { method: string; params: unknown }[]
  syncs:      (string[] | undefined)[]
  log:        string[]
  syncFails?: boolean
}

function deps (scenario: Scenario): Parameters<typeof applyChanges>[0] {
  return {
    workspaceRoot: '/work',
    engine:        {
      request<R> (method: string, params?: unknown): Promise<R> {
        scenario.requests.push({ method, params })

        return Promise.resolve((method === WATCH_ROUTE_METHOD ? scenario.route : { runs: scenario.runs ?? [] }) as R)
      },
    },
    sync: async scope => {
      scenario.syncs.push(scope)
      if (scenario.syncFails) {
        throw new Error('confluence is down')
      }
    },
    log: line => { scenario.log.push(line) },
  }
}

const scenario = (route: Partial<WatchRouteResult>, rest: Partial<Scenario> = {}): Scenario =>
  ({ route: { everything: false, generators: [], markdown: [], ...route }, requests: [], syncs: [], log: [], ...rest })

describe('applyChanges', () => {
  it('syncs just the changed pages and runs no generator for a Markdown edit', async () => {
    const run = scenario({ markdown: ['docs/guide.md'] })

    const applied = await applyChanges(deps(run), ['docs/guide.md'])

    expect(run.requests.map(request => request.method)).toEqual([WATCH_ROUTE_METHOD])
    expect(run.requests[0].params).toEqual({ workspaceRoot: '/work', changed: ['docs/guide.md'] })
    expect(run.syncs).toEqual([['docs/guide.md']])
    expect(applied.touched).toEqual(['docs/guide.md'])
    expect(run.log).toEqual(['syncing docs/guide.md'])
  })

  it('regenerates only the routed generators, then syncs what they wrote with the changed pages', async () => {
    const run = scenario(
      { generators: [1], markdown: ['README.md'] },
      { runs: [{ index: 1, type: 'go-docs', output: 'docs/api', written: ['docs/api/core.md', 'docs/api/data.json'], unchanged: ['docs/api/README.md'] }] },
    )

    const applied = await applyChanges(deps(run), ['libs/core/a.go', 'README.md'])

    expect(run.requests[1]).toEqual({ method: GENERATORS_RUN_METHOD, params: { workspaceRoot: '/work', generators: [1] } })
    expect(run.syncs).toEqual([['README.md', 'docs/api/core.md']])
    expect(applied.touched).toEqual(['docs/api/core.md', 'docs/api/data.json', 'README.md'])
    expect(run.log).toEqual(['go-docs → docs/api: 2 written, 1 unchanged, 0 removed', 'syncing README.md, docs/api/core.md'])
  })

  it('syncs nothing when the changes touch no page and write none', async () => {
    const run = scenario({ generators: [0] }, { runs: [{ index: 0, type: 'go-docs', output: 'docs/api', unchanged: ['docs/api/README.md'] }] })

    await applyChanges(deps(run), ['libs/core/a.go'])

    expect(run.syncs).toEqual([])
    expect(run.log.at(-1)).toBe('nothing to sync for libs/core/a.go')
  })

  it('syncs the rest when a generator fails, and says so', async () => {
    const run = scenario(
      { generators: [0, 1], markdown: ['README.md'] },
      { runs: [{ index: 0, type: 'ts-docs', output: 'docs/ts', error: 'TypeDoc was not found' }, { index: 1, type: 'go-docs', output: 'docs/api', written: ['docs/api/a.md'] }] },
    )

    await applyChanges(deps(run), ['README.md', 'a.ts', 'a.go'])

    expect(run.log[0]).toBe('ts-docs → docs/ts: failed: TypeDoc was not found')
    expect(run.syncs).toEqual([['README.md', 'docs/api/a.md']])
  })

  it('syncs everything when the settings changed', async () => {
    const run = scenario({ everything: true, generators: [0] }, { runs: [{ index: 0, type: 'go-docs', output: 'docs/api', written: ['docs/api/a.md'] }] })

    await applyChanges(deps(run), ['.lore-master.yaml'])

    expect(run.syncs).toEqual([undefined])
    expect(run.log).toContain('settings changed: syncing everything')
  })

  it('names a few files and counts the rest', async () => {
    const markdown = ['a.md', 'b.md', 'c.md', 'd.md', 'e.md']
    const run = scenario({ markdown })

    await applyChanges(deps(run), markdown)

    expect(run.log).toEqual(['syncing a.md, b.md, c.md and 2 more'])
  })

  it('lets a failed sync reach the caller, so the batch is tried again', async () => {
    const run = scenario({ markdown: ['README.md'] }, { syncFails: true })

    await expect(applyChanges(deps(run), ['README.md'])).rejects.toThrow('confluence is down')
  })
})
