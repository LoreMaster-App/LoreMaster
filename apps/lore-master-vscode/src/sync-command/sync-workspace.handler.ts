import * as vscode from 'vscode'

/** The command id contributed in package.json. */
export const SYNC_WORKSPACE_COMMAND = 'loreMaster.syncWorkspace'

/**
 * Syncs the open workspace to its documentation platform.
 *
 * The skeleton only answers: the engine process (#59), the connection (#60), the
 * sync target (#61) and the sync itself (#62) arrive in their own issues.
 */
export async function syncWorkspace (): Promise<void> {
  await vscode.window.showInformationMessage('Lore Master: syncing is not available yet.')
}
