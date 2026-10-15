import type { CredentialVault } from './credential-vault.client'
import type { ConnectionMeta, ConnectionStore } from './secret-store.client'

export interface VaultSyncResult {
  /** Connections that were only in the keychain and are now in this editor profile. */
  imported: number
  /** Connections that were only in this editor profile and are now in the keychain. */
  exported: number
}

const sameSite = (a: string, b: string): boolean => a.replace(/\/+$/, '').toLowerCase() === b.replace(/\/+$/, '').toLowerCase()

/**
 * Makes the editor profile's connections and the keychain's the same set, in both directions:
 * what the keychain has and this profile lacks is added here (so a new profile signs in without
 * a token being entered again), and what this profile has and the keychain lacks is remembered
 * there (so the command line and other profiles can use it). A connection both have is left
 * alone. Nothing is ever deleted: forgetting is an explicit remove.
 */
export async function syncWithVault (store: ConnectionStore, vault: CredentialVault): Promise<VaultSyncResult> {
  const remembered = await vault.list()
  const local = store.list()
  let imported = 0
  let exported = 0

  for (const entry of remembered) {
    if (local.some(connection => sameSite(connection.baseUrl, entry.baseUrl))) {
      continue
    }
    const meta: ConnectionMeta = { baseUrl: entry.baseUrl, edition: entry.edition ?? '', displayName: entry.displayName ?? '', user: entry.user ?? '' }
    await store.add(meta, entry.credential)
    imported += 1
  }

  for (const connection of local) {
    if (remembered.some(entry => sameSite(entry.baseUrl, connection.baseUrl))) {
      continue
    }
    const credential = await store.credential(connection.baseUrl)
    if (!credential) {
      continue
    }
    await vault.store({ baseUrl: connection.baseUrl, edition: connection.edition, displayName: connection.displayName, user: connection.user, credential })
    exported += 1
  }

  return { imported, exported }
}
