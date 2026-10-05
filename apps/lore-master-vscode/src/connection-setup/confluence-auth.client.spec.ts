import type { Credential } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { createConfluenceAuthProvider } from './confluence-auth.client'

function fakeStore (metas: ConnectionMeta[], credentials: Record<string, Credential> = {}): ConnectionStore & { removed: string[] } {
  const removed: string[] = []
  let connections = [...metas]

  return {
    removed,
    list:       () => connections,
    credential: baseUrl => Promise.resolve(credentials[baseUrl]),
    async add (meta, credential) {
      connections = [...connections.filter(connection => connection.baseUrl !== meta.baseUrl), meta]
      credentials[meta.baseUrl] = credential
    },
    async remove (baseUrl) {
      removed.push(baseUrl)
      connections = connections.filter(connection => connection.baseUrl !== baseUrl)
    },
  }
}

function meta (baseUrl: string): ConnectionMeta {
  return { baseUrl, edition: 'datacenter', displayName: 'Ada', user: 'ada' }
}

describe('createConfluenceAuthProvider', () => {
  it('surfaces stored connections as Accounts-menu sessions', async () => {
    const provider = createConfluenceAuthProvider({
      store:  fakeStore([meta('https://dc.example')], { 'https://dc.example': { kind: 'pat', token: 'at-1' } }),
      signIn: () => Promise.resolve(undefined),
    })

    const sessions = await provider.getSessions(undefined, {})
    expect(sessions).toHaveLength(1)
    expect(sessions[0]).toMatchObject({ id: 'https://dc.example', accessToken: 'at-1', account: { label: 'Ada (https://dc.example)' } })
  })

  it('creates a session by running the sign-in flow, firing an added event', async () => {
    const store = fakeStore([])
    const added: string[] = []
    const provider = createConfluenceAuthProvider({
      store,
      signIn: async () => {
        await store.add(meta('https://dc.example'), { kind: 'pat', token: 'at-1' })

        return meta('https://dc.example')
      },
    })
    provider.onDidChangeSessions(event => { added.push(...(event.added ?? []).map(session => session.id)) })

    const session = await provider.createSession([], {})
    expect(session.id).toBe('https://dc.example')
    expect(session.accessToken).toBe('at-1')
    expect(added).toEqual(['https://dc.example'])
  })

  it('rejects when the sign-in is cancelled', async () => {
    const provider = createConfluenceAuthProvider({ store: fakeStore([]), signIn: () => Promise.resolve(undefined) })

    await expect(provider.createSession([], {})).rejects.toThrow('cancelled')
  })

  it('removes the connection and fires a removed event on sign-out', async () => {
    const store = fakeStore([meta('https://dc.example')], { 'https://dc.example': { kind: 'pat', token: 't' } })
    const removed: string[] = []
    const provider = createConfluenceAuthProvider({ store, signIn: () => Promise.resolve(undefined) })
    provider.onDidChangeSessions(event => { removed.push(...(event.removed ?? []).map(session => session.id)) })

    await provider.removeSession('https://dc.example')
    expect(store.removed).toEqual(['https://dc.example'])
    expect(removed).toEqual(['https://dc.example'])
  })
})
