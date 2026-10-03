import type { ConnectionMeta, ConnectionStore, Credential } from '../secret-storage'
import {
  type AuthMethod,
  authMethodsFor,
  type ConnectionUI,
  type EngineRequester,
  setUpConnection,
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

// A UI with every prompt answered; override per test.
function scriptedUI (over: Partial<ConnectionUI> = {}): ConnectionUI & { errors: string[]; connected: ConnectionMeta[]; offered: AuthMethod[][] } {
  const errors: string[] = []
  const connected: ConnectionMeta[] = []
  const offered: AuthMethod[][] = []

  return {
    errors,
    connected,
    offered,
    promptBaseUrl:  () => Promise.resolve('https://x.atlassian.net/wiki'),
    pickAuthMethod: methods => {
      offered.push(methods)

      return Promise.resolve(methods[0])
    },
    promptCredential: () => Promise.resolve(apitoken),
    showError:        message => {
      errors.push(message)

      return Promise.resolve()
    },
    showConnected: meta => {
      connected.push(meta)

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

function engineOf (handlers: Record<string, (params: unknown) => unknown>): EngineRequester {
  return { request: (method, params) => Promise.resolve(handlers[method](params) as never) }
}

describe('authMethodsFor', () => {
  it('offers only what each edition accepts', () => {
    expect(authMethodsFor('cloud')).toEqual(['apitoken'])
    expect(authMethodsFor('datacenter')).toEqual(['pat'])
    expect(authMethodsFor('server')).toEqual(['pat', 'basic'])
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

    expect(ui.offered).toEqual([['apitoken']])
    expect(store.added).toEqual([{ meta: { baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'Ed', user: 'acc-1' }, credential: apitoken }])
    expect(closed).toEqual(['s1']) // the verify session is thrown away
    expect(ui.connected).toHaveLength(1)
    expect(meta?.displayName).toBe('Ed')
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

  it('ends quietly when the URL prompt is cancelled, touching the engine not at all', async () => {
    let called = false
    const engine: EngineRequester = {
      request: () => {
        called = true

        return Promise.resolve(undefined as never)
      },
    }

    const meta = await setUpConnection({ engine, store: recordingStore(), ui: scriptedUI({ promptBaseUrl: () => Promise.resolve(undefined) }) })

    expect(called).toBe(false)
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
})
