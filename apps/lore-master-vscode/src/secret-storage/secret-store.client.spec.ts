import type * as vscode from 'vscode'
import { type ConnectionMeta, createConnectionStore } from './secret-store.client'
import type { Credential } from '../engine-protocol'

function fakeSecrets (): vscode.SecretStorage {
  const store: Record<string, string> = {}

  return {
    get:   key => Promise.resolve(store[key]),
    store: (key, value) => {
      store[key] = value

      return Promise.resolve()
    },
    delete: key => {
      delete store[key]

      return Promise.resolve()
    },
    keys:        () => Promise.resolve(Object.keys(store)),
    onDidChange: (() => ({ dispose () {} })) as vscode.SecretStorage['onDidChange'],
  }
}

function fakeMemento (): vscode.Memento {
  const store: Record<string, unknown> = {}

  return {
    keys:   () => Object.keys(store),
    get:    <T>(key: string, value?: T) => (Object.hasOwn(store, key) ? store[key] : value) as T,
    update: (key, value) => {
      store[key] = value

      return Promise.resolve()
    },
  }
}

const meta: ConnectionMeta = { baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'Ed', user: 'acc-1' }
const credential: Credential = { kind: 'apitoken', email: 'ed@x.com', token: 'secret' }

describe('createConnectionStore', () => {
  it('stores the credential in SecretStorage and the metadata in the memento', async () => {
    const secrets = fakeSecrets()
    const store = createConnectionStore(secrets, fakeMemento())

    await store.add(meta, credential)

    expect(store.list()).toEqual([meta])
    expect(await store.credential(meta.baseUrl)).toEqual(credential)
    // Keyed by base URL.
    expect(await secrets.get('loreMaster.credential.https://x.atlassian.net/wiki')).toBe(JSON.stringify(credential))
  })

  it('replaces an existing connection for the same base URL rather than duplicating it', async () => {
    const store = createConnectionStore(fakeSecrets(), fakeMemento())

    await store.add(meta, credential)
    await store.add({ ...meta, displayName: 'Eduardo' }, { kind: 'pat', token: 't2' })

    expect(store.list()).toHaveLength(1)
    expect(store.list()[0].displayName).toBe('Eduardo')
    expect(await store.credential(meta.baseUrl)).toEqual({ kind: 'pat', token: 't2' })
  })

  it('forgets both the credential and the metadata on remove', async () => {
    const store = createConnectionStore(fakeSecrets(), fakeMemento())
    await store.add(meta, credential)

    await store.remove(meta.baseUrl)

    expect(store.list()).toEqual([])
    expect(await store.credential(meta.baseUrl)).toBeUndefined()
  })

  it('returns undefined for an unknown connection', async () => {
    const store = createConnectionStore(fakeSecrets(), fakeMemento())

    expect(await store.credential('https://nope')).toBeUndefined()
    expect(store.list()).toEqual([])
  })
})
