import type { EngineClient } from '../engine-process'
import {
  type Edition,
  type Output,
  PAGES_PUBLISH_METHOD,
  SESSION_OPEN_METHOD,
  SETTINGS_READ_METHOD,
  SETTINGS_SAVE_METHOD,
  SYNC_EXECUTE_METHOD,
  SYNC_PLAN_METHOD,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { createTargetStore } from '../sync-target'
import { syncOutputs } from './sync-outputs.use-case'
import type { SyncUI } from './sync-run.use-case'

type Route = (params: unknown) => unknown

interface FakeEngine extends EngineClient {
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
    onNotification () {
      return { dispose () {} }
    },
    onRequest () {
      return { dispose () {} }
    },
    dispose () {},
  }
}

const meta: ConnectionMeta = { baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'Ed', user: 'acc' }

function connectionStore (): ConnectionStore {
  return {
    list:       () => [meta],
    credential: () => Promise.resolve({ kind: 'apitoken', email: 'e', token: 't' }),
    add:        () => Promise.resolve(),
    remove:     () => Promise.resolve(),
  }
}

function buildUI (over: Partial<SyncUI> = {}): SyncUI & { errors: string[] } {
  const errors: string[] = []

  return {
    errors,
    pickSpace:         spaces => Promise.resolve(spaces[0]),
    pickAction:        () => Promise.resolve({ kind: 'use' }),
    promptSearchQuery: () => Promise.resolve(undefined),
    pickSearchResult:  () => Promise.resolve(undefined),
    promptTitlePrefix: value => Promise.resolve(value),
    choose:            () => Promise.resolve('run'),
    writeDetails:      () => {},
    showErrors:        () => Promise.resolve(),
    pickConnection:    metas => Promise.resolve(metas[0]),
    noConnections:     () => Promise.resolve(),
    pickStorageTypes:  () => Promise.resolve(undefined),
    pickOutputs:       () => Promise.resolve(undefined),
    promptRepo:        () => Promise.resolve(undefined),
    promptBranch:      () => Promise.resolve(undefined),
    withProgress:      (_title, task) => task(() => {}),
    report:            () => {},
    status:            () => {},
    confirmForce:      () => Promise.resolve(false),
    error:             message => {
      errors.push(message)

      return Promise.resolve()
    },
    ...over,
  }
}

const session = { sessionId: 's', baseUrl: meta.baseUrl, edition: 'cloud' as Edition, user: { displayName: 'Ed' } }

const confluenceOutput: Output = {
  platform:       'confluence',
  baseUrl:        meta.baseUrl,
  space:          'ENG',
  parentPageId:   'home',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
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

const blankConfluenceScaffold: Output = { ...confluenceOutput, baseUrl: '', space: '', parentPageId: '', titlePrefix: '' }

function deps (engine: EngineClient, ui: SyncUI, options = {}): Parameters<typeof syncOutputs>[0] {
  return { engine, connections: connectionStore(), targets: createTargetStore(), workspaceRoot: '/w', ui, ...options }
}

const bothConfigured = { exists: true, firstSync: false, settings: { version: 1, outputs: [confluenceOutput, pagesOutput] } }

describe('syncOutputs', () => {
  it('fans out over every configured output: Confluence is planned, GitHub Pages is published', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => bothConfigured,
      [SESSION_OPEN_METHOD]:  () => session,
      [SYNC_PLAN_METHOD]:     () => ({ planId: 'p', actions: [], counts: {} }),
      [SYNC_EXECUTE_METHOD]:  () => ({ pages: [] }),
      [PAGES_PUBLISH_METHOD]: () => ({ changed: true, files: 2, branch: 'gh-pages' }),
    })

    await syncOutputs(deps(engine, buildUI()))

    expect((engine.calls.find(call => call.method === SYNC_PLAN_METHOD)?.params as { output: number }).output).toBe(0)
    expect((engine.calls.find(call => call.method === PAGES_PUBLISH_METHOD)?.params as { output: number }).output).toBe(1)
  })

  it('runs the first-run setup and saves the chosen outputs when none are configured', async () => {
    let saved: unknown
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: false, firstSync: true, settings: { version: 1, outputs: [blankConfluenceScaffold] } }),
      [SETTINGS_SAVE_METHOD]: params => {
        saved = params

        return null
      },
      [PAGES_PUBLISH_METHOD]: () => ({ changed: true, files: 1, branch: 'gh-pages' }),
    })
    const ui = buildUI({ pickStorageTypes: () => Promise.resolve(['github-pages']), promptRepo: () => Promise.resolve(''), promptBranch: () => Promise.resolve('gh-pages') })

    await syncOutputs(deps(engine, ui))

    const outputs = (saved as { settings: { outputs: Output[] } }).settings.outputs
    expect(outputs).toHaveLength(1)
    expect(outputs[0].platform).toBe('github-pages')
    expect((engine.calls.find(call => call.method === PAGES_PUBLISH_METHOD)?.params as { output: number }).output).toBe(0)
  })

  it('syncs only the chosen subset for "sync to…"', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => bothConfigured,
      [PAGES_PUBLISH_METHOD]: () => ({ changed: true, files: 1, branch: 'gh-pages' }),
    })
    const ui = buildUI({ pickOutputs: () => Promise.resolve([1]) })

    await syncOutputs(deps(engine, ui), { choose: true })

    expect(engine.calls.some(call => call.method === SYNC_PLAN_METHOD)).toBe(false)
    expect((engine.calls.find(call => call.method === PAGES_PUBLISH_METHOD)?.params as { output: number }).output).toBe(1)
  })

  it('syncs only Confluence outputs for the current-file sync', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => bothConfigured,
      [SESSION_OPEN_METHOD]:  () => session,
      [SYNC_PLAN_METHOD]:     () => ({ planId: 'p', actions: [], counts: {} }),
      [SYNC_EXECUTE_METHOD]:  () => ({ pages: [] }),
    })

    await syncOutputs(deps(engine, buildUI()), { confluenceOnly: true })

    expect(engine.calls.some(call => call.method === SYNC_PLAN_METHOD)).toBe(true)
    expect(engine.calls.some(call => call.method === PAGES_PUBLISH_METHOD)).toBe(false)
  })
})
