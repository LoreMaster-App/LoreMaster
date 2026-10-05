import type { EngineClient } from '../engine-process'
import {
  type Edition,
  PAGE_CHILDREN_METHOD,
  SESSION_OPEN_METHOD,
  SPACE_LIST_METHOD,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { configureConfluence } from './configure-confluence.use-case'
import type { ConfluenceSetupUI } from './set-up-storages.use-case'

type Route = (params: unknown) => unknown

function fakeEngine (routes: Record<string, Route>): EngineClient {
  return {
    request: (method, params) => {
      const route: Route | undefined = routes[method]

      return Promise.resolve((route ? route(params) : null) as never)
    },
    onNotification: () => ({ dispose () {} }),
    onRequest:      () => ({ dispose () {} }),
    dispose () {},
  }
}

const meta: ConnectionMeta = { baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'Ed', user: 'acc' }

function connectionStore (metas: ConnectionMeta[] = [meta]): ConnectionStore {
  return {
    list:       () => metas,
    credential: () => Promise.resolve({ kind: 'apitoken', email: 'e', token: 't' }),
    add:        () => Promise.resolve(),
    remove:     () => Promise.resolve(),
  }
}

interface Recording {
  errors:           string[]
  toldNoConnection: boolean
}

function buildUI (over: Partial<ConfluenceSetupUI> = {}): ConfluenceSetupUI & { recording: Recording } {
  const recording: Recording = { errors: [], toldNoConnection: false }

  return {
    recording,
    pickSpace:         spaces => Promise.resolve(spaces[0]),
    pickAction:        () => Promise.resolve({ kind: 'use' }),
    promptSearchQuery: () => Promise.resolve(undefined),
    pickSearchResult:  () => Promise.resolve(undefined),
    promptTitlePrefix: value => Promise.resolve(value),
    pickConnection:    metas => Promise.resolve(metas[0]),
    noConnections:     () => {
      recording.toldNoConnection = true

      return Promise.resolve()
    },
    error: message => {
      recording.errors.push(message)

      return Promise.resolve()
    },
    ...over,
  }
}

const session = { sessionId: 's', baseUrl: meta.baseUrl, edition: 'cloud' as Edition, user: { displayName: 'Ed' } }

describe('configureConfluence', () => {
  it('builds a configured output from the connection, space, parent and prefix', async () => {
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]:  () => session,
      [SPACE_LIST_METHOD]:    () => ({ spaces: [{ id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }] }),
      [PAGE_CHILDREN_METHOD]: () => ({ pages: [] }),
    })

    const output = await configureConfluence({ engine, connections: connectionStore(), ui: buildUI() })

    expect(output).toMatchObject({
      platform:       'confluence',
      baseUrl:        meta.baseUrl,
      space:          'ENG',
      parentPageId:   'home',
      titlePrefix:    'Engineering',
      direction:      'to-platform',
      mermaidMode:    'image',
      titleCollision: 'fail',
      linkMode:       'title',
    })
  })

  it('tells the user when there is no connection and no way to add one', async () => {
    const ui = buildUI()
    const output = await configureConfluence({ engine: fakeEngine({}), connections: connectionStore([]), ui })

    expect(output).toBeUndefined()
    expect(ui.recording.toldNoConnection).toBe(true)
  })

  it('launches Add Connection inline when there is none, then continues', async () => {
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]:  () => session,
      [SPACE_LIST_METHOD]:    () => ({ spaces: [{ id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }] }),
      [PAGE_CHILDREN_METHOD]: () => ({ pages: [] }),
    })
    const ui = buildUI()

    const output = await configureConfluence({ engine, connections: connectionStore([]), ui, addConnection: () => Promise.resolve(meta) })

    expect(ui.recording.toldNoConnection).toBe(false)
    expect(output).toMatchObject({ platform: 'confluence', baseUrl: meta.baseUrl, space: 'ENG' })
  })

  it('returns undefined when the space pick is cancelled', async () => {
    const engine = fakeEngine({
      [SESSION_OPEN_METHOD]: () => session,
      [SPACE_LIST_METHOD]:   () => ({ spaces: [{ id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }] }),
    })

    const output = await configureConfluence({ engine, connections: connectionStore(), ui: buildUI({ pickSpace: () => Promise.resolve(undefined) }) })

    expect(output).toBeUndefined()
  })
})
