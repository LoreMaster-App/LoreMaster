import type { EngineClient } from '../engine-process'
import {
  CREDENTIAL_DELETE_METHOD,
  CREDENTIAL_LIST_METHOD,
  CREDENTIAL_STORE_METHOD,
  type CredentialListResult,
  type VaultEntry,
} from '../engine-protocol'

/** The operating system's keychain as the extension reaches it: through the engine. */
export interface CredentialVault {
  store (entry: VaultEntry): Promise<void>
  delete (baseUrl: string): Promise<void>
  list (): Promise<VaultEntry[]>
}

/** A {@link CredentialVault} over the engine's credential/* methods. */
export function createEngineVault (engine: Pick<EngineClient, 'request'>): CredentialVault {
  return {
    async store (entry) {
      await engine.request(CREDENTIAL_STORE_METHOD, { entry })
    },
    async delete (baseUrl) {
      await engine.request(CREDENTIAL_DELETE_METHOD, { baseUrl })
    },
    async list () {
      const result = await engine.request<CredentialListResult>(CREDENTIAL_LIST_METHOD)

      return result.entries
    },
  }
}
