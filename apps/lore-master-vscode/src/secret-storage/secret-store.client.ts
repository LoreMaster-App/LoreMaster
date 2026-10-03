import type * as vscode from 'vscode'

/** How a user signs in. Mirrors the engine's `Credential`; `kind` decides the fields. */
export type CredentialKind = 'apitoken' | 'pat' | 'basic'

/** A credential kept in the OS secret store, never in settings or logs. */
export interface Credential {
  kind:      CredentialKind
  email?:    string
  token?:    string
  user?:     string
  password?: string
}

/** The non-secret facts about a connection, shown in lists. Kept in `globalState`. */
export interface ConnectionMeta {
  baseUrl:     string
  edition:     string
  displayName: string
  /** The account id (Cloud) or username (Data Center/Server) the credential signs in as. */
  user:        string
}

/** Stored connections: metadata in `globalState`, the credential in `SecretStorage`. */
export interface ConnectionStore {
  list (): ConnectionMeta[]
  credential (baseUrl: string): Promise<Credential | undefined>
  add (meta: ConnectionMeta, credential: Credential): Promise<void>
  remove (baseUrl: string): Promise<void>
}

const CONNECTIONS_KEY = 'loreMaster.connections'

const credentialKey = (baseUrl: string): string => `loreMaster.credential.${baseUrl}`

/**
 * A {@link ConnectionStore} over VS Code's `SecretStorage` (the credential) and a
 * `Memento` such as `globalState` (the metadata). The two are kept in step: `add` and
 * `remove` write both, keyed by the normalised base URL, so a base URL has at most one
 * credential and one metadata row.
 */
export function createConnectionStore (secrets: vscode.SecretStorage, memento: vscode.Memento): ConnectionStore {
  const list = (): ConnectionMeta[] => memento.get<ConnectionMeta[]>(CONNECTIONS_KEY, [])

  return {
    list,

    async credential (baseUrl) {
      const raw = await secrets.get(credentialKey(baseUrl))

      return raw === undefined ? undefined : JSON.parse(raw) as Credential
    },

    async add (meta, credential) {
      await secrets.store(credentialKey(meta.baseUrl), JSON.stringify(credential))
      const others = list().filter(connection => connection.baseUrl !== meta.baseUrl)
      await memento.update(CONNECTIONS_KEY, [...others, meta])
    },

    async remove (baseUrl) {
      await secrets.delete(credentialKey(baseUrl))
      await memento.update(CONNECTIONS_KEY, list().filter(connection => connection.baseUrl !== baseUrl))
    },
  }
}
