import { describedConfig } from '../../test/config-description/config-description.fixture'
import { type ConfigDescription, type Credential, type CredentialKindDescriptor, CONFIG_DESCRIBE_METHOD } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import {
  authMethodsFor,
  type ConnectionUI,
  type EngineRequester,
  setUpConnection,
  type SitePrompt,
} from './connection-setup.handler'

function recordingStore (): ConnectionStore & { added: { meta: ConnectionMeta; credential: Credential }[] } {
  const added: { meta: ConnectionMeta; credential: Credential }[] = []

  return {
    added,
    list:       () => added.map(entry => entry.meta),
    credential: () => Promise.resolve(undefined),
    add:        (meta, credential) => {
      added.push({ meta, credential })

      return Promise.resolve()
    },
    remove: () => Promise.resolve(),
  }
}

const apitoken: Credential = { kind: 'apitoken', email: 'ed@x.com', token: 'good' }

interface Recorded {
  errors:    string[]
  connected: ConnectionMeta[]
  offered:   CredentialKindDescriptor[][]
  sites:     SitePrompt[]
}

// A UI with every prompt answered; override per test.
function scriptedUI (over: Partial<ConnectionUI> = {}): ConnectionUI & Recorded {
  const recorded: Recorded = { errors: [], connected: [], offered: [], sites: [] }

  return {
    ...recorded,
    pickPlatform:  platforms => Promise.resolve(platforms[0]),
    promptBaseUrl: site => {
      recorded.sites.push(site)

      return Promise.resolve('https://x.atlassian.net/wiki')
    },
    pickAuthMethod: methods => {
      recorded.offered.push(methods)

      return Promise.resolve(methods[0])
    },
    promptCredential: () => Promise.resolve(apitoken),
    showError:        message => {
      recorded.errors.push(message)

      return Promise.resolve()
    },
    showConnected: meta => {
      recorded.connected.push(meta)

      return Promise.resolve()
    },
    ...over,
  }
}

const sessionResult = {
  sessionId: 's1',
  baseUrl:   'https://x.atlassian.net/wiki',
  edition:   'cloud',
  user:      { displayName: 'Ed', accountId: 'acc-1' },
}

function engineOf (handlers: Record<string, (params: unknown) => unknown>, description: ConfigDescription = describedConfig()): EngineRequester {
  const all: Record<string, (params: unknown) => unknown> = { ...handlers, [CONFIG_DESCRIBE_METHOD]: () => description }

  return { request: (method, params) => Promise.resolve(all[method](params) as never) }
}

const confluence = (): NonNullable<ReturnType<typeof describedConfig>['platforms'][number]['connection']> => {
  const connection = describedConfig().platforms.find(platform => platform.id === 'confluence')?.connection
  if (!connection) {
    throw new Error('the catalog describes no Confluence sign-in')
  }

  return connection
}

describe('authMethodsFor', () => {
  it('offers only what each edition accepts, from the platform\'s own description', () => {
    expect(authMethodsFor(confluence(), 'cloud').map(method => method.id)).toEqual(['apitoken'])
    expect(authMethodsFor(confluence(), 'datacenter').map(method => method.id)).toEqual(['pat'])
    expect(authMethodsFor(confluence(), 'server').map(method => method.id)).toEqual(['pat', 'basic'])
  })

  it('offers a method that names no editions for every one', () => {
    const connection = { field: 'baseUrl', kinds: [{ id: 'key', label: 'API key', fields: [{ key: 'token', label: 'Key', env: 'API_KEY', secret: true }] }] }

    expect(authMethodsFor(connection, 'anything').map(method => method.id)).toEqual(['key'])
  })
})

describe('setUpConnection', () => {
  it('detects the edition, verifies, stores, and reports the connection', async () => {
    const closed: string[] = []
    const engine = engineOf({
      'edition/detect': () => ({ baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud' }),
      'session/open':   () => sessionResult,
      'session/close':  (params) => {
        closed.push((params as { sessionId: string }).sessionId)

        return null
      },
    })
    const store = recordingStore()
    const ui = scriptedUI()

    const meta = await setUpConnection({ engine, store, ui })

    expect(store.added).toEqual([{ meta: { baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'Ed', user: 'acc-1' }, credential: apitoken }])
    expect(closed).toEqual(['s1']) // the verify session is thrown away
    expect(ui.connected).toHaveLength(1)
    expect(meta?.displayName).toBe('Ed')
  })

  it('asks for the address as the description words it, and does not ask which method when only one fits', async () => {
    const engine = engineOf({
      'edition/detect': () => ({ baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud' }),
      'session/open':   () => sessionResult,
      'session/close':  () => null,
    })
    const ui = scriptedUI()

    await setUpConnection({ engine, store: recordingStore(), ui })

    expect(ui.sites).toEqual([expect.objectContaining({ platform: 'Confluence', label: 'Confluence address', placeholder: 'https://example.atlassian.net/wiki' })])
    expect(ui.offered).toEqual([])
  })

  it('asks which method when the edition accepts more than one', async () => {
    const engine = engineOf({
      'edition/detect': () => ({ baseUrl: 'https://wiki.acme.com', edition: 'server' }),
      'session/open':   () => ({ ...sessionResult, edition: 'server' }),
      'session/close':  () => null,
    })
    const ui = scriptedUI()

    await setUpConnection({ engine, store: recordingStore(), ui })

    expect(ui.offered.map(methods => methods.map(method => method.id))).toEqual([['pat', 'basic']])
  })

  it('shows the engine\'s refusal verbatim and stores nothing on a bad credential', async () => {
    const engine = engineOf({
      'edition/detect': () => ({ baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud' }),
      'session/open':   () => { throw new Error('The site refused the credential.') },
    })
    const store = recordingStore()
    const ui = scriptedUI()

    const meta = await setUpConnection({ engine, store, ui })

    expect(ui.errors).toEqual(['The site refused the credential.'])
    expect(store.added).toEqual([])
    expect(meta).toBeUndefined()
  })

  it('shows a detection failure verbatim and never asks for a credential', async () => {
    let askedCredential = false
    const engine = engineOf({
      'edition/detect': () => { throw new Error('site is not reachable') },
      'session/open':   () => sessionResult,
    })
    const ui = scriptedUI({
      promptCredential: () => {
        askedCredential = true

        return Promise.resolve(apitoken)
      },
    })

    const meta = await setUpConnection({ engine, store: recordingStore(), ui })

    expect(ui.errors).toEqual(['site is not reachable'])
    expect(askedCredential).toBe(false)
    expect(meta).toBeUndefined()
  })

  it('ends quietly when the address prompt is cancelled, asking the engine only what can be configured', async () => {
    const asked: string[] = []
    const engine: EngineRequester = {
      request: (method) => {
        asked.push(method)

        return Promise.resolve(describedConfig() as never)
      },
    }

    const meta = await setUpConnection({ engine, store: recordingStore(), ui: scriptedUI({ promptBaseUrl: () => Promise.resolve(undefined) }) })

    expect(asked).toEqual([CONFIG_DESCRIBE_METHOD])
    expect(meta).toBeUndefined()
  })

  it('still stores the connection if closing the verify session fails', async () => {
    const engine = engineOf({
      'edition/detect': () => ({ baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud' }),
      'session/open':   () => sessionResult,
      'session/close':  () => { throw new Error('already gone') },
    })
    const store = recordingStore()

    const meta = await setUpConnection({ engine, store, ui: scriptedUI() })

    expect(store.added).toHaveLength(1)
    expect(meta).toBeDefined()
  })

  it('connects to a platform it was never taught, with the sign-in its description lists', async () => {
    const description = describedConfig()
    description.platforms = [{
      id:              'notion',
      label:           'Notion',
      description:     '',
      icon:            'book',
      delivery:        'pages',
      nameTemplate:    '',
      summaryTemplate: '',
      connection:      { field: 'workspaceUrl', kinds: [{ id: 'apitoken', label: 'Integration token', fields: [{ key: 'token', label: 'Token', env: 'NOTION_TOKEN', secret: true }] }] },
      fields:          [{ key: 'workspaceUrl', label: 'Workspace address', type: 'text' }],
    }]
    const engine = engineOf({ 'session/open': () => sessionResult, 'session/close': () => null }, description)
    const ui = scriptedUI()

    const meta = await setUpConnection({ engine, store: recordingStore(), ui })

    expect(ui.sites[0]).toMatchObject({ platform: 'Notion', label: 'Workspace address' })
    expect(meta?.displayName).toBe('Ed')
  })

  it('says so when a platform has no sign-in method for the edition detected', async () => {
    const engine = engineOf({ 'edition/detect': () => ({ baseUrl: 'https://x', edition: 'mainframe' }) })
    const ui = scriptedUI()

    const meta = await setUpConnection({ engine, store: recordingStore(), ui })

    expect(ui.errors[0]).toContain('no sign-in method')
    expect(meta).toBeUndefined()
  })
})
