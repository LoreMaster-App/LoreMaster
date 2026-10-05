import {
  type Output,
  PAGES_PUBLISH_METHOD,
  SETTINGS_READ_METHOD,
  SETTINGS_SAVE_METHOD,
} from '../engine-protocol'
import { type PagesEngine, type PagesUI, publishPages } from './publish-pages.use-case'

type Route = (params: unknown) => unknown

interface FakeEngine extends PagesEngine {
  calls: { method: string; params: unknown }[]
}

function fakeEngine (routes: Record<string, Route>): FakeEngine {
  const calls: { method: string; params: unknown }[] = []

  return {
    calls,
    request (method, params) {
      calls.push({ method, params })
      const route: Route | undefined = routes[method]

      return Promise.resolve((route ? route(params) : null) as never)
    },
  }
}

function buildUI (): PagesUI & { reported: string[][]; statuses: string[]; errors: string[] } {
  const reported: string[][] = []
  const statuses: string[] = []
  const errors: string[] = []

  return {
    reported,
    statuses,
    errors,
    withProgress: (_title, task) => task(),
    report:       lines => { reported.push(lines) },
    status:       message => { statuses.push(message) },
    error:        message => {
      errors.push(message)

      return Promise.resolve()
    },
  }
}

const pagesOutput: Output = {
  platform:       'github-pages',
  baseUrl:        '',
  space:          '',
  parentPageId:   '',
  titlePrefix:    '',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    '',
  titleCollision: '',
  linkMode:       '',
  repo:           '',
  branch:         '',
}

const confluenceOutput: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}

describe('publishPages', () => {
  it('publishes the existing github-pages output and reports the result', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [pagesOutput] } }),
      [PAGES_PUBLISH_METHOD]: () => ({ changed: true, files: 4, branch: 'gh-pages', commit: 'abc1234', url: 'https://me.github.io/repo/' }),
    })
    const ui = buildUI()

    await publishPages({ engine, workspaceRoot: '/w', ui })

    const publish = engine.calls.find(call => call.method === PAGES_PUBLISH_METHOD)
    expect(publish?.params).toEqual({ workspaceRoot: '/w', output: 0 })
    expect(engine.calls.some(call => call.method === SETTINGS_SAVE_METHOD)).toBe(false)
    expect(ui.reported[0][0]).toContain('Published 4 file(s) to gh-pages (abc1234). https://me.github.io/repo/')
    expect(ui.statuses[0]).toContain('https://me.github.io/repo/')
  })

  it('adds a github-pages output on a fresh workspace, replacing the scaffold', async () => {
    let saved: unknown
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: false, firstSync: true, settings: { version: 1, outputs: [confluenceOutput] } }),
      [SETTINGS_SAVE_METHOD]: params => {
        saved = params

        return null
      },
      [PAGES_PUBLISH_METHOD]: () => ({ changed: true, files: 2, branch: 'gh-pages', commit: 'def' }),
    })
    const ui = buildUI()

    await publishPages({ engine, workspaceRoot: '/w', ui })

    // The scaffold is replaced by a single github-pages output, and that index is published.
    expect((saved as { settings: { outputs: Output[] } }).settings.outputs).toEqual([pagesOutput])
    expect(engine.calls.find(call => call.method === PAGES_PUBLISH_METHOD)?.params).toEqual({ workspaceRoot: '/w', output: 0 })
  })

  it('appends a github-pages output beside a configured Confluence one', async () => {
    let saved: unknown
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [confluenceOutput] } }),
      [SETTINGS_SAVE_METHOD]: params => {
        saved = params

        return null
      },
      [PAGES_PUBLISH_METHOD]: () => ({ changed: true, files: 1, branch: 'gh-pages' }),
    })

    await publishPages({ engine, workspaceRoot: '/w', ui: buildUI() })

    expect((saved as { settings: { outputs: Output[] } }).settings.outputs).toEqual([confluenceOutput, pagesOutput])
    expect(engine.calls.find(call => call.method === PAGES_PUBLISH_METHOD)?.params).toEqual({ workspaceRoot: '/w', output: 1 })
  })

  it('shows the Markdown errors and does not report a publish', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [pagesOutput] } }),
      [PAGES_PUBLISH_METHOD]: () => ({ changed: false, files: 0, errors: ['readme.md: no H1'] }),
    })
    const ui = buildUI()

    await publishPages({ engine, workspaceRoot: '/w', ui })

    expect(ui.reported[0].join('\n')).toContain('Not published')
    expect(ui.reported[0].join('\n')).toContain('readme.md: no H1')
    expect(ui.statuses[0]).toContain('not published')
  })

  it('reports a no-op when nothing changed', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [pagesOutput] } }),
      [PAGES_PUBLISH_METHOD]: () => ({ changed: false, files: 3, branch: 'gh-pages' }),
    })
    const ui = buildUI()

    await publishPages({ engine, workspaceRoot: '/w', ui })

    expect(ui.reported[0][0]).toContain('already up to date')
  })

  it('surfaces an engine failure as an error', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [pagesOutput] } }),
      [PAGES_PUBLISH_METHOD]: () => { throw new Error('git push failed: permission denied') },
    })
    const ui = buildUI()

    await publishPages({ engine, workspaceRoot: '/w', ui })

    expect(ui.errors[0]).toContain('git push failed')
  })
})
