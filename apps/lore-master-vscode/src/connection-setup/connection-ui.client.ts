import * as vscode from 'vscode'
import type { Credential } from '../engine-protocol'
import type { ConnectionMeta } from '../secret-storage'
import type { AuthMethod, ConnectionUI } from './connection-setup.handler'

const AUTH_METHOD_LABELS: Record<AuthMethod, string> = {
  apitoken: 'Email and API token (Cloud)',
  pat:      'Personal access token (Data Center / Server)',
  basic:    'Username and password (Server)',
  // OAuth is interactive (a browser sign-in), not a pasted secret, so it is offered by a
  // separate flow — not listed here yet. The label keeps the map type-complete.
  oauth:    'OAuth sign-in (Data Center)',
}

/** The {@link ConnectionUI} backed by VS Code's input boxes and QuickPick. */
export function createConnectionUI (): ConnectionUI {
  return {
    promptBaseUrl () {
      return Promise.resolve(vscode.window.showInputBox({
        title:          'LoreMaster: add a connection',
        prompt:         'Confluence site URL',
        placeHolder:    'https://your-site.atlassian.net/wiki',
        ignoreFocusOut: true,
      }))
    },

    async pickAuthMethod (methods) {
      const picked = await vscode.window.showQuickPick(
        methods.map(method => ({ label: AUTH_METHOD_LABELS[method], method })),
        { title: 'LoreMaster: sign-in method', placeHolder: 'How do you sign in?' },
      )

      return picked?.method
    },

    async promptCredential (method) {
      if (method === 'oauth') {
        // OAuth collects the admin's incoming-link client id and, optionally, the scope the
        // link was configured with; the browser sign-in that follows produces the tokens.
        const clientId = await ask('OAuth client id (from the site admin)')
        if (clientId === undefined) {
          return
        }
        const scope = await ask('OAuth scope (optional; e.g. WRITE)')
        if (scope === undefined) {
          return
        }

        return { kind: 'oauth', clientId, ...(scope && { scope }) } satisfies Credential
      }
      if (method === 'apitoken') {
        const email = await ask('Atlassian account email')
        if (email === undefined) {
          return
        }
        const token = await ask('API token', true)

        return token === undefined ? undefined : { kind: 'apitoken', email, token }
      }
      if (method === 'pat') {
        const token = await ask('Personal access token', true)

        return token === undefined ? undefined : { kind: 'pat', token }
      }
      const user = await ask('Username')
      if (user === undefined) {
        return
      }
      const password = await ask('Password', true)

      return password === undefined ? undefined : { kind: 'basic', user, password } satisfies Credential
    },

    async showError (message) {
      await vscode.window.showErrorMessage(message)
    },

    async showConnected (meta: ConnectionMeta) {
      await vscode.window.showInformationMessage(`LoreMaster: connected to ${meta.baseUrl} as ${meta.displayName}.`)
    },
  }
}

function ask (prompt: string, password = false): Thenable<string | undefined> {
  return vscode.window.showInputBox({ prompt, password, ignoreFocusOut: true })
}
