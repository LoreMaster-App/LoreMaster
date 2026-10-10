import type { EngineClient } from '../engine-process'
import { PAGES_CHECK_METHOD, type PagesCheckResult } from '../engine-protocol'
import { checkPages } from './check-pages.use-case'

function engine (answer: PagesCheckResult | Error, calls: { method: string; params: unknown }[] = []): EngineClient {
  return {
    async request (method: string, params?: unknown) {
      calls.push({ method, params })
      if (answer instanceof Error) {
        throw answer
      }

      return answer as never
    },
    onNotification: () => ({ dispose () {} }),
    dispose () {},
  } as unknown as EngineClient
}

describe('checkPages', () => {
  it('asks pages/check for the output and says the branch is up to date', async () => {
    const calls: { method: string; params: unknown }[] = []
    const outcome = await checkPages(engine({ upToDate: true, changesTotal: 0, files: 9, branch: 'gh-pages' }, calls), '/w', 1)

    expect(calls).toEqual([{ method: PAGES_CHECK_METHOD, params: { workspaceRoot: '/w', output: 1 } }])
    expect(outcome).toEqual({ ok: true, upToDate: true, message: 'gh-pages is up to date (9 files).' })
  })

  it('names the first changes and counts the rest', async () => {
    const changes = Array.from({ length: 7 }, (_, index) => ({ path: `p${index}.html`, kind: 'modified' }))
    const outcome = await checkPages(engine({ upToDate: false, changesTotal: 7, files: 9, branch: 'gh-pages', changes }), '/w', 0)

    expect(outcome.ok && outcome.upToDate).toBe(false)
    expect(outcome.ok && outcome.message).toBe('gh-pages is out of date: 7 files would change (modified p0.html, modified p1.html, modified p2.html, modified p3.html, modified p4.html, and 2 more).')
  })

  it('reports Markdown errors as the reason it could not compare', async () => {
    const outcome = await checkPages(engine({ upToDate: false, changesTotal: 0, files: 0, errors: ['a.md: no H1', 'b.md: no H1'] }), '/w', 0)

    expect(outcome).toEqual({ ok: false, reason: 'the site would not build: a.md: no H1 (and 1 more)' })
  })

  it('returns the engine error, such as a branch that holds another site', async () => {
    const outcome = await checkPages(engine(new Error('clone failed')), '/w', 0)

    expect(outcome).toEqual({ ok: false, reason: 'clone failed' })
  })
})
