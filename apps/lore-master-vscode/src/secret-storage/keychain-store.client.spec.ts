import type { Credential, VaultEntry } from '../engine-protocol'
import type { CredentialVault } from './credential-vault.client'
import { withKeychain } from './keychain-store.client'
import type { ConnectionMeta, ConnectionStore } from './secret-store.client'

const meta: ConnectionMeta = { baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'Ed', user: 'acc-1' }
const credential: Credential = { kind: 'apitoken', email: 'ed@x.com', token: 'secret' }

function memoryStore (): ConnectionStore & { metas: ConnectionMeta[], credentials: Map<string, Credential> } {
  const metas: ConnectionMeta[] = []
  const credentials = new Map<string, Credential>()

  return {
    metas,
    credentials,
    list:       () => metas,
    credential: baseUrl => Promise.resolve(credentials.get(baseUrl)),
    add (added, secret) {
      metas.push(added)
      credentials.set(added.baseUrl, secret)

      return Promise.resolve()
    },
    remove (baseUrl) {
      credentials.delete(baseUrl)
      metas.splice(0, metas.length, ...metas.filter(each => each.baseUrl !== baseUrl))

      return Promise.resolve()
    },
  }
}

function memoryVault (options: { failing?: boolean } = {}): CredentialVault & { entries: VaultEntry[] } {
  const entries: VaultEntry[] = []

  return {
    entries,
    store (entry) {
      if (options.failing) {
        return Promise.reject(new Error('no keychain'))
      }
      entries.push(entry)

      return Promise.resolve()
    },
    delete (baseUrl) {
      if (options.failing) {
        return Promise.reject(new Error('no keychain'))
      }
      entries.splice(0, entries.length, ...entries.filter(each => each.baseUrl !== baseUrl))

      return Promise.resolve()
    },
    list: () => Promise.resolve(entries),
  }
}

describe('withKeychain', () => {
  it('remembers a connection in the keychain too, and forgets it there on removal', async () => {
    const vault = memoryVault()
    const store = withKeychain(memoryStore(), vault, { enabled: () => true, onError () {} })

    await store.add(meta, credential)
    expect(vault.entries).toEqual([{ baseUrl: meta.baseUrl, edition: 'cloud', displayName: 'Ed', user: 'acc-1', credential }])

    await store.remove(meta.baseUrl)
    expect(vault.entries).toEqual([])
  })

  it('leaves the keychain alone while the option is off', async () => {
    const vault = memoryVault()
    const inner = memoryStore()
    const store = withKeychain(inner, vault, { enabled: () => false, onError () {} })

    await store.add(meta, credential)

    expect(inner.metas).toEqual([meta])
    expect(vault.entries).toEqual([])
  })

  it('keeps the connection and reports when the keychain cannot be reached', async () => {
    const errors: unknown[] = []
    const inner = memoryStore()
    const store = withKeychain(inner, memoryVault({ failing: true }), { enabled: () => true, onError: error => { errors.push(error) } })

    await store.add(meta, credential)

    expect(inner.metas).toEqual([meta])
    expect(errors).toHaveLength(1)
  })
})
