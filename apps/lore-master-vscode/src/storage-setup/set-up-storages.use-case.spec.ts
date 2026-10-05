import type { EngineClient } from '../engine-process'
import {
  type Edition,
  PAGE_CHILDREN_METHOD,
  SESSION_OPEN_METHOD,
  SPACE_LIST_METHOD,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { setUpStorages, type StorageSetupUI, type StorageType } from './set-up-storages.use-case'

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

function connectionStore (): ConnectionStore {
  return {
    list:       () => [meta],
    credential: () => Promise.resolve({ kind: 'apitoken', email: 'e', token: 't' }),
    add:        () => Promise.resolve(),
    remove:     () => Promise.resolve(),
  }
}

const session = { sessionId: 's', baseUrl: meta.baseUrl, edition: 'cloud' as Edition, user: { displayName: 'Ed' } }

function buildUI (types: StorageType[] | undefined): StorageSetupUI {
  return {
    pickStorageTypes:  () => Promise.resolve(types),
    pickSpace:         spaces => Promise.resolve(spaces[0]),
    pickAction:        () => Promise.resolve({ kind: 'use' }),
    promptSearchQuery: () => Promise.resolve(undefined),
    pickSearchResult:  () => Promise.resolve(undefined),
    promptTitlePrefix: value => Promise.resolve(value),
    pickConnection:    metas => Promise.resolve(metas[0]),
    noConnections:     () => Promise.resolve(),
    error:             () => Promise.resolve(),
    promptRepo:        () => Promise.resolve('owner/name'),
    promptBranch:      () => Promise.resolve('gh-pages'),
  }
}

const engine = (): EngineClient => fakeEngine({
  [SESSION_OPEN_METHOD]:  () => session,
  [SPACE_LIST_METHOD]:    () => ({ spaces: [{ id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }] }),
  [PAGE_CHILDREN_METHOD]: () => ({ pages: [] }),
})

describe('setUpStorages', () => {
  it('configures each chosen storage type and returns the outputs to save', async () => {
    const outputs = await setUpStorages({ engine: engine(), connections: connectionStore(), ui: buildUI(['confluence', 'github-pages']) })

    expect(outputs).toHaveLength(2)
    expect(outputs?.map(output => output.platform)).toEqual(['confluence', 'github-pages'])
  })

  it('returns undefined when no type is chosen', async () => {
    expect(await setUpStorages({ engine: engine(), connections: connectionStore(), ui: buildUI(undefined) })).toBeUndefined()
    expect(await setUpStorages({ engine: engine(), connections: connectionStore(), ui: buildUI([]) })).toBeUndefined()
  })
})
