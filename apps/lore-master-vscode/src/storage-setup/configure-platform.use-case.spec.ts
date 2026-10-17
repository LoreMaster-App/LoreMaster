import { describedConfig } from '../../test/config-description/config-description.fixture'
import type { EngineClient } from '../engine-process'
import {
  type ConfigDescription,
  type Edition,
  type FieldDescriptor,
  PAGE_CHILDREN_METHOD,
  SESSION_OPEN_METHOD,
  SPACE_LIST_METHOD,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { configurePlatform } from './configure-platform.use-case'
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
  asked:            string[]
}

function buildUI (over: Partial<StorageSetupUI> = {}): StorageSetupUI & { recording: Recording } {
  const recording: Recording = { errors: [], toldNoConnection: false, asked: [] }

  return {
    recording,
    pickStorageTypes:  () => Promise.resolve(undefined),
    pickSpace:         spaces => Promise.resolve(spaces[0]),
    pickAction:        () => Promise.resolve({ kind: 'use' }),
    promptSearchQuery: () => Promise.resolve(undefined),
    pickSearchResult:  () => Promise.resolve(undefined),
    promptTitlePrefix: value => Promise.resolve(value),
    promptField:       (field, initial) => {
      recording.asked.push(field.key)

      return Promise.resolve(initial ?? '')
    },
    pickConnection: metas => Promise.resolve(metas[0]),
    noConnections:  () => {
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

const routes = {
  [SESSION_OPEN_METHOD]:  () => session,
  [SPACE_LIST_METHOD]:    () => ({ spaces: [{ id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }] }),
  [PAGE_CHILDREN_METHOD]: () => ({ pages: [] }),
}

function platformOf (description: ConfigDescription, id: string) {
  const platform = description.platforms.find(each => each.id === id)
  if (!platform) {
    throw new Error(`no ${id}`)
  }

  return platform
}

describe('configurePlatform', () => {
  const description = describedConfig()
  const confluence = platformOf(description, 'confluence')

  it('builds a configured Confluence output from the connection, space, parent and prefix', async () => {
    const ui = buildUI()

    const output = await configurePlatform({ engine: fakeEngine(routes), connections: connectionStore(), ui, description }, confluence)

    expect(output).toMatchObject({
      platform:       'confluence',
      baseUrl:        meta.baseUrl,
      space:          'ENG',
      parentPageId:   'home',
      direction:      'to-platform',
      mermaidMode:    'image',
      titleCollision: 'fail',
      linkMode:       'title',
    })
    expect(ui.recording.asked).toEqual(['titlePrefix'])
  })

  it('offers the title prefix the parent page fills in, as the description says', async () => {
    const offered: Array<string | undefined> = []
    const ui = buildUI({
      promptField: (_field, initial) => {
        offered.push(initial)

        return Promise.resolve(initial)
      },
    })

    const output = await configurePlatform({ engine: fakeEngine(routes), connections: connectionStore(), ui, description }, confluence)

    expect(offered).toHaveLength(1)
    expect(output?.titlePrefix).toBe(offered[0])
  })

  it('tells the user when there is no connection and no way to add one', async () => {
    const ui = buildUI()
    const output = await configurePlatform({ engine: fakeEngine({}), connections: connectionStore([]), ui, description }, confluence)

    expect(output).toBeUndefined()
    expect(ui.recording.toldNoConnection).toBe(true)
  })

  it('launches Add Connection inline when there is none, then continues', async () => {
    const ui = buildUI()

    const output = await configurePlatform({ engine: fakeEngine(routes), connections: connectionStore([]), ui, description, addConnection: () => Promise.resolve(meta) }, confluence)

    expect(ui.recording.toldNoConnection).toBe(false)
    expect(output).toMatchObject({ platform: 'confluence', baseUrl: meta.baseUrl, space: 'ENG' })
  })

  it('returns undefined when the space pick is cancelled', async () => {
    const output = await configurePlatform({ engine: fakeEngine(routes), connections: connectionStore(), ui: buildUI({ pickSpace: () => Promise.resolve(undefined) }), description }, confluence)

    expect(output).toBeUndefined()
  })

  it('returns undefined when a prompt is cancelled', async () => {
    const output = await configurePlatform({ engine: fakeEngine(routes), connections: connectionStore(), ui: buildUI({ promptField: () => Promise.resolve(undefined) }), description }, confluence)

    expect(output).toBeUndefined()
  })

  it('needs no sign-in for a site: it asks for the repository and keeps the other defaults', async () => {
    const ui = buildUI({ promptField: () => Promise.resolve(' owner/name ') })

    const output = await configurePlatform({ engine: fakeEngine({}), connections: connectionStore([]), ui, description }, platformOf(description, 'github-pages'))

    expect(output).toMatchObject({ platform: 'github-pages', repo: 'owner/name', branch: '', baseUrl: '' })
    expect(ui.recording.toldNoConnection).toBe(false)
  })

  it('configures a platform it was never taught, from its description alone', async () => {
    const added = describedConfig()
    const workspace: FieldDescriptor = { key: 'workspace', label: 'Workspace', type: 'text', required: true }
    added.platforms.push({ id: 'notion', label: 'Notion', description: '', icon: 'book', delivery: 'pages', nameTemplate: '', summaryTemplate: '', fields: [workspace] })
    const ui = buildUI({ promptField: field => Promise.resolve(field.key === 'workspace' ? 'acme' : undefined) })

    const output = await configurePlatform({ engine: fakeEngine({}), connections: connectionStore([]), ui, description: added }, platformOf(added, 'notion'))

    expect(output).toMatchObject({ platform: 'notion', workspace: 'acme' })
  })
})
