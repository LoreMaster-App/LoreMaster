import * as vscode from 'vscode'
import type { Credential } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'

/** The id and label VS Code shows for Lore Master's Confluence accounts. */
export const CONFLUENCE_AUTH_ID = 'lore-master-confluence'
export const CONFLUENCE_AUTH_LABEL = 'Confluence (Lore Master)'

/** What the provider needs: the connection store, and a way to run the sign-in flow. */
export interface AuthProviderDeps {
  store:  ConnectionStore
  /** Runs the connection flow and resolves to the connection it stored, or undefined when
   *  the user cancelled. In the extension this is setUpConnection. */
  signIn: () => Promise<ConnectionMeta | undefined>
}

/** The access token to expose for a session. OAuth sessions carry their access token;
 *  other kinds carry their token or password, so VS Code has a non-empty value. */
function tokenOf (credential: Credential | undefined): string {
  return credential?.accessToken ?? credential?.token ?? credential?.password ?? 'confluence'
}

function sessionOf (meta: ConnectionMeta, accessToken: string): vscode.AuthenticationSession {
  return {
    id:      meta.baseUrl,
    accessToken,
    account: { id: meta.user || meta.baseUrl, label: `${meta.displayName} (${meta.baseUrl})` },
    scopes:  [],
  }
}

/**
 * A vscode.authentication provider that surfaces the stored Confluence connections in the
 * Accounts menu: sign in (runs the connection flow), see the account, and sign out (removes
 * the connection). The connection store stays the source of truth — getSessions reads it
 * live, so a connection added through the command appears here too — and this only adds the
 * Accounts-menu way in and out.
 */
export function createConfluenceAuthProvider (deps: AuthProviderDeps): vscode.AuthenticationProvider & { dispose (): void } {
  const changes = new vscode.EventEmitter<vscode.AuthenticationProviderAuthenticationSessionsChangeEvent>()
  const sessionFor = async (meta: ConnectionMeta): Promise<vscode.AuthenticationSession> =>
    sessionOf(meta, tokenOf(await deps.store.credential(meta.baseUrl)))

  return {
    onDidChangeSessions: changes.event,

    getSessions () {
      return Promise.all(deps.store.list().map(meta => sessionFor(meta)))
    },

    async createSession () {
      const meta = await deps.signIn()
      if (!meta) {
        throw new Error('Confluence sign-in was cancelled')
      }
      const session = await sessionFor(meta)
      changes.fire({ added: [session], removed: [], changed: [] })

      return session
    },

    async removeSession (sessionId) {
      const meta = deps.store.list().find(connection => connection.baseUrl === sessionId)
      await deps.store.remove(sessionId)
      if (meta) {
        changes.fire({ added: [], removed: [sessionOf(meta, '')], changed: [] })
      }
    },

    dispose () {
      changes.dispose()
    },
  }
}

/** Registers the provider with VS Code and returns a disposable that unregisters it. */
export function registerConfluenceAuth (deps: AuthProviderDeps): { dispose (): void } {
  const provider = createConfluenceAuthProvider(deps)
  const registration = vscode.authentication.registerAuthenticationProvider(
    CONFLUENCE_AUTH_ID, CONFLUENCE_AUTH_LABEL, provider, { supportsMultipleAccounts: true },
  )

  return {
    dispose () {
      registration.dispose()
      provider.dispose()
    },
  }
}
