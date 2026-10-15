import type { CredentialVault } from './credential-vault.client'
import type { ConnectionStore } from './secret-store.client'

/**
 * A {@link ConnectionStore} that also remembers what it is given in the operating system's
 * keychain, and forgets it there too, while the option is on. A keychain that cannot be reached
 * is reported and never fails the connection: the editor profile has the credential either way.
 */
export function withKeychain (store: ConnectionStore, vault: CredentialVault, options: { enabled: () => boolean, onError: (error: unknown) => void }): ConnectionStore {
  return {
    list:       () => store.list(),
    credential: baseUrl => store.credential(baseUrl),

    async add (meta, credential) {
      await store.add(meta, credential)
      if (!options.enabled()) {
        return
      }
      try {
        await vault.store({ baseUrl: meta.baseUrl, edition: meta.edition, displayName: meta.displayName, user: meta.user, credential })
      } catch (error) {
        options.onError(error)
      }
    },

    async remove (baseUrl) {
      await store.remove(baseUrl)
      if (!options.enabled()) {
        return
      }
      try {
        await vault.delete(baseUrl)
      } catch (error) {
        options.onError(error)
      }
    },
  }
}
