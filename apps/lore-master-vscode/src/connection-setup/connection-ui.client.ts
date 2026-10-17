import * as vscode from 'vscode'
import type { Credential, CredentialKindDescriptor } from '../engine-protocol'
import type { ConnectionMeta } from '../secret-storage'
import type { ConnectionUI } from './connection-setup.handler'

/** The {@link ConnectionUI} backed by VS Code's input boxes and QuickPick. What it says comes from the engine's description. */
export function createConnectionUI (): ConnectionUI {
  return {
    async pickPlatform (platforms) {
      const picked = await vscode.window.showQuickPick(
        platforms.map(platform => ({ label: platform.label, description: platform.description, platform })),
        { title: 'LoreMaster: add a connection', placeHolder: 'Which platform?' },
      )

      return picked?.platform
    },

    promptBaseUrl (site) {
      return Promise.resolve(vscode.window.showInputBox({
        title:          `LoreMaster: add a connection to ${site.platform}`,
        prompt:         site.help ?? site.label,
        placeHolder:    site.placeholder,
        ignoreFocusOut: true,
      }))
    },

    async pickAuthMethod (methods) {
      const picked = await vscode.window.showQuickPick(
        methods.map(method => ({ label: method.label, description: method.help, method })),
        { title: 'LoreMaster: sign-in method', placeHolder: 'How do you sign in?' },
      )

      return picked?.method
    },

    async promptCredential (method) {
      return askFields(method)
    },

    async showError (message) {
      await vscode.window.showErrorMessage(message)
    },

    async showConnected (meta: ConnectionMeta) {
      await vscode.window.showInformationMessage(`LoreMaster: connected to ${meta.baseUrl} as ${meta.displayName}.`)
    },
  }
}

/** Asks for each thing the sign-in method wants (a secret is masked); undefined when the user backs out of any. */
async function askFields (method: CredentialKindDescriptor): Promise<Credential | undefined> {
  const answers: Record<string, string> = {}
  for (const field of method.fields) {
    const answer = await vscode.window.showInputBox({ title: method.label, prompt: field.label, password: field.secret === true, ignoreFocusOut: true })
    if (answer === undefined) {
      return undefined
    }
    answers[field.key] = answer
  }

  return { kind: method.id, ...answers } as Credential
}
