import type { EngineClient } from '../engine-process'
import {
  type Edition,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  SETTINGS_READ_METHOD,
  SETTINGS_SAVE_METHOD,
  SYNC_EXECUTE_METHOD,
  SYNC_PLAN_METHOD,
} from '../engine-protocol'
import type { Credential } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { createTargetStore } from '../sync-target'
import { type ProgressStep, runSync, type SyncUI } from './sync-run.use-case'

type Route = (params: unknown) => unknown

interface FakeEngine extends EngineClient {
  calls: { method: string; params: unknown }[]
}

function fakeEngine (routes: Record<string, Route>): FakeEngine {
  const calls: { method: string; params: unknown }[] = []
  let progress: ((params: unknown) => void) | undefined

  return {
    calls,
    request (method, params) {
      calls.push({ method, params })
      if (method === SYNC_EXECUTE_METHOD) {
        progress?.({ planId: (params as { planId: string }).planId, message: 'writing', done: 1, total: 2 })
      }

      const route: Route | undefined = routes[method]

      return Promise.resolve((route ? route(params) : null) as never)
    },
    onNotification (_method, handler) {
      progress = handler

      return { dispose () { progress = undefined } }
    },
    onRequest () {
      return { dispose () {} }
    },
    dispose () {},
  }
}

const meta: ConnectionMeta = { baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'Ed', user: 'acc' }
const credential: Credential = { kind: 'apitoken', email: 'e', token: 't' }

function connectionStore (metas: ConnectionMeta[] = [meta], cred: Credential | undefined = credential): ConnectionStore {
  return {
    list:       () => metas,
    credential: () => Promise.resolve(cred),
    add:        () => Promise.resolve(),
    remove:     () => Promise.resolve(),
  }
}

function buildUI (over: Partial<SyncUI> = {}): SyncUI & { reported: string[][]; steps: ProgressStep[]; statuses: string[]; errors: string[] } {
  const reported: string[][] = []
  const steps: ProgressStep[] = []
  const statuses: string[] = []
  const errors: string[] = []

  return {
    reported,
    steps,
    statuses,
    errors,
    pickSpace:         spaces => Promise.resolve(spaces[0]),
    pickAction:        () => Promise.resolve({ kind: 'use' }),
    promptSearchQuery: () => Promise.resolve(undefined),
    pickSearchResult:  () => Promise.resolve(undefined),
    promptTitlePrefix: value => Promise.resolve(value),
    choose:            () => Promise.resolve('run'),
    writeDetails:      () => {},
    showErrors:        list => {
      errors.push(...list)

      return Promise.resolve()
    },
    pickConnection: metas => Promise.resolve(metas[0]),
    noConnections:  () => Promise.resolve(),
    withProgress:   (_title, task) => task(step => { steps.push(step) }),
    report:         lines => { reported.push(lines) },
    status:         message => { statuses.push(message) },
    confirmForce:   () => Promise.resolve(false),
    error:          message => {
      errors.push(message)

      return Promise.resolve()
    },
    ...over,
  }
}

const session = { sessionId: 's', baseUrl: meta.baseUrl, edition: 'cloud' as Edition, user: { displayName: 'Ed' } }

const configuredOutput = {
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

describe('runSync', () => {
  it('syncs a configured folder end to end, reporting progress, outcomes and closing the session', async () => {
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]:  () => session,
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [configuredOutput] } }),
      [SYNC_PLAN_METHOD]:     () => ({ planId: 'p', actions: [{ kind: 'create', title: 'A' }, { kind: 'create', title: 'B' }], counts: { create: 2 } }),
      [SYNC_EXECUTE_METHOD]:  () => ({ pages: [{ title: 'A', planned: 'create', outcome: 'written' }, { title: 'B', planned: 'create', outcome: 'written' }] }),
    })
    const ui = buildUI()

    await runSync({ engine, connections: connectionStore(), targets: createTargetStore(), workspaceRoot: '/w', ui })

    expect(ui.steps).toEqual([{ message: 'writing', done: 1, total: 2 }])
    expect(ui.reported[0]).toEqual(['written   A', 'written   B'])
    expect(ui.statuses).toEqual(['Lore Master: synced 2 page(s)'])
    expect(engine.calls.some(call => call.method === SETTINGS_SAVE_METHOD)).toBe(false)
    expect(engine.calls.at(-1)?.method).toBe(SESSION_CLOSE_METHOD)
  })

  it('runs the first-sync pickers and saves the output when the folder has none', async () => {
    let saved: unknown
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]:  () => session,
      // A fresh workspace is read back with one unconfigured scaffold output, as the engine returns.
      [SETTINGS_READ_METHOD]: () => ({ exists: false, firstSync: true, settings: { version: 1, outputs: [{ ...configuredOutput, baseUrl: '', space: '', parentPageId: '', titlePrefix: '' }] } }),
      'space/list':           () => ({ spaces: [{ id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }] }),
      'page/children':        () => ({ pages: [] }),
      [SETTINGS_SAVE_METHOD]: params => {
        saved = params

        return null
      },
      [SYNC_PLAN_METHOD]:    () => ({ planId: 'p', actions: [], counts: { create: 1 } }),
      [SYNC_EXECUTE_METHOD]: () => ({ pages: [{ title: 'A', planned: 'create', outcome: 'written' }] }),
    })

    await runSync({ engine, connections: connectionStore(), targets: createTargetStore(), workspaceRoot: '/w', ui: buildUI() })

    // The scaffold is filled in place, not duplicated: exactly one, fully configured, output.
    expect((saved as { settings: { outputs: unknown[] } }).settings.outputs).toHaveLength(1)
    expect(saved).toMatchObject({ workspaceRoot: '/w', settings: { outputs: [{ baseUrl: meta.baseUrl, space: 'ENG', parentPageId: 'home', titlePrefix: 'Engineering' }] } })
  })

  it('stops before executing when the plan has errors', async () => {
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]:  () => session,
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [configuredOutput] } }),
      [SYNC_PLAN_METHOD]:     () => ({ planId: 'p', actions: [], counts: {}, errors: ['missing H1 in a.md'] }),
    })
    const ui = buildUI()

    await runSync({ engine, connections: connectionStore(), targets: createTargetStore(), workspaceRoot: '/w', ui })

    expect(ui.errors).toEqual(['missing H1 in a.md'])
    expect(engine.calls.some(call => call.method === SYNC_EXECUTE_METHOD)).toBe(false)
  })

  it('does not execute when the preview is cancelled', async () => {
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]:  () => session,
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [configuredOutput] } }),
      [SYNC_PLAN_METHOD]:     () => ({ planId: 'p', actions: [], counts: { create: 1 } }),
    })

    await runSync({ engine, connections: connectionStore(), targets: createTargetStore(), workspaceRoot: '/w', ui: buildUI({ choose: () => Promise.resolve('cancel') }) })

    expect(engine.calls.some(call => call.method === SYNC_EXECUTE_METHOD)).toBe(false)
  })

  it('offers a forced re-run on conflict and runs it when confirmed', async () => {
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]:  () => session,
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [configuredOutput] } }),
      [SYNC_PLAN_METHOD]:     () => ({ planId: 'p', actions: [{ kind: 'conflict', title: 'A' }], counts: { conflict: 1 } }),
      [SYNC_EXECUTE_METHOD]:  () => ({ pages: [{ title: 'A', planned: 'conflict', outcome: 'skipped' }] }),
    })

    await runSync({ engine, connections: connectionStore(), targets: createTargetStore(), workspaceRoot: '/w', ui: buildUI({ confirmForce: () => Promise.resolve(true) }) })

    const executes = engine.calls.filter(call => call.method === SYNC_EXECUTE_METHOD)
    expect(executes).toHaveLength(2)
    expect((executes[1].params as { force: boolean }).force).toBe(true)
  })

  it('does nothing but prompt when there is no connection', async () => {
    const engine = fakeEngine({})
    let told = false

    await runSync({
      engine,
      connections:   connectionStore([]),
      targets:       createTargetStore(),
      workspaceRoot: '/w',
      ui:            buildUI({
        noConnections: () => {
          told = true

          return Promise.resolve()
        },
      }),
    })

    expect(told).toBe(true)
    expect(engine.calls).toEqual([])
  })
})
