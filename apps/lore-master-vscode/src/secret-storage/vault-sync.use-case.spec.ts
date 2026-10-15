import type { Credential, VaultEntry } from '../engine-protocol'
import type { CredentialVault } from './credential-vault.client'
import type { ConnectionMeta, ConnectionStore } from './secret-store.client'
import { syncWithVault } from './vault-sync.use-case'

const credentialA: Credential = { kind: 'apitoken', email: 'a@x.com', token: 'a' }
const credentialB: Credential = { kind: 'pat', token: 'b' }

function storeWith (initial: [ConnectionMeta, Credential][]): ConnectionStore {
  const metas = initial.map(([meta]) => meta)
  const credentials = new Map(initial.map(([meta, credential]) => [meta.baseUrl, credential]))

  return {
    list:       () => metas,
    credential: baseUrl => Promise.resolve(credentials.get(baseUrl)),
    add (meta, credential) {
      metas.push(meta)
      credentials.set(meta.baseUrl, credential)

      return Promise.resolve()
    },
    remove: () => Promise.resolve(),
  }
}

function vaultWith (entries: VaultEntry[]): CredentialVault & { entries: VaultEntry[] } {
  return {
    entries,
    store (entry) {
      entries.push(entry)

      return Promise.resolve()
    },
    delete: () => Promise.resolve(),
    list:   () => Promise.resolve(entries),
  }
}

const here: [ConnectionMeta, Credential] = [{ baseUrl: 'https://a.atlassian.net/wiki', edition: 'cloud', displayName: 'A', user: 'u-a' }, credentialA]

describe('syncWithVault', () => {
  it('brings in what only the keychain has, so a new profile needs no token', async () => {
    const store = storeWith([])
    const vault = vaultWith([{ baseUrl: 'https://b.example/confluence', edition: 'datacenter', displayName: 'B', user: 'bob', credential: credentialB }])

    const result = await syncWithVault(store, vault)

    expect(result).toEqual({ imported: 1, exported: 0 })
    expect(store.list()).toEqual([{ baseUrl: 'https://b.example/confluence', edition: 'datacenter', displayName: 'B', user: 'bob' }])
    expect(await store.credential('https://b.example/confluence')).toEqual(credentialB)
  })

  it('remembers what only this profile has, for the command line and other profiles', async () => {
    const store = storeWith([here])
    const vault = vaultWith([])

    const result = await syncWithVault(store, vault)

    expect(result).toEqual({ imported: 0, exported: 1 })
    expect(vault.entries).toEqual([{ baseUrl: here[0].baseUrl, edition: 'cloud', displayName: 'A', user: 'u-a', credential: credentialA }])
  })

  it('leaves a connection both have alone, whatever the case or trailing slash', async () => {
    const store = storeWith([here])
    const vault = vaultWith([{ baseUrl: 'HTTPS://A.atlassian.net/wiki/', credential: credentialA }])

    const result = await syncWithVault(store, vault)

    expect(result).toEqual({ imported: 0, exported: 0 })
    expect(store.list()).toHaveLength(1)
  })

  it('skips a local connection that has lost its credential', async () => {
    const store = storeWith([])
    await store.add(here[0], credentialA)
    const emptied: ConnectionStore = { ...store, credential: () => Promise.resolve(undefined) }

    const result = await syncWithVault(emptied, vaultWith([]))

    expect(result.exported).toBe(0)
  })
})
