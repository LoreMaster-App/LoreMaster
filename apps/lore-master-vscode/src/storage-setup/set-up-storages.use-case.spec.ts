import { describedConfig } from '../../test/config-description/config-description.fixture'
import type { EngineClient } from '../engine-process'
import {
  type Edition,
  PAGE_CHILDREN_METHOD,
  SESSION_OPEN_METHOD,
  SPACE_LIST_METHOD,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { setUpStorages } from './set-up-storages.use-case'
import type { StorageSetupUI } from './storage-setup.contract'

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

function buildUI (types: string[] | undefined): StorageSetupUI & { offered: string[][] } {
  const offered: string[][] = []

  return {
    offered,
    pickStorageTypes: platforms => {
      offered.push(platforms.map(platform => platform.id))

      return Promise.resolve(types)
    },
    pickSpace:         spaces => Promise.resolve(spaces[0]),
    pickAction:        () => Promise.resolve({ kind: 'use' }),
    promptSearchQuery: () => Promise.resolve(undefined),
    pickSearchResult:  () => Promise.resolve(undefined),
    promptTitlePrefix: value => Promise.resolve(value),
    promptField:       (field, initial) => Promise.resolve(field.key === 'repo' ? 'owner/name' : initial ?? ''),
    pickConnection:    metas => Promise.resolve(metas[0]),
    noConnections:     () => Promise.resolve(),
    error:             () => Promise.resolve(),
  }
}

const engine = (): EngineClient => fakeEngine({
  [SESSION_OPEN_METHOD]:  () => session,
  [SPACE_LIST_METHOD]:    () => ({ spaces: [{ id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }] }),
  [PAGE_CHILDREN_METHOD]: () => ({ pages: [] }),
})

const description = describedConfig()

describe('setUpStorages', () => {
  it('offers the platforms the engine describes, configures each chosen one and returns the outputs to save', async () => {
    const ui = buildUI(['confluence', 'github-pages'])

    const outputs = await setUpStorages({ engine: engine(), connections: connectionStore(), ui, description })

    expect(ui.offered).toEqual([['confluence', 'github-pages', 'github-wiki']])
    expect(outputs?.map(output => output.platform)).toEqual(['confluence', 'github-pages'])
  })

  it('returns undefined when no type is chosen', async () => {
    expect(await setUpStorages({ engine: engine(), connections: connectionStore(), ui: buildUI(undefined), description })).toBeUndefined()
    expect(await setUpStorages({ engine: engine(), connections: connectionStore(), ui: buildUI([]), description })).toBeUndefined()
  })
})
