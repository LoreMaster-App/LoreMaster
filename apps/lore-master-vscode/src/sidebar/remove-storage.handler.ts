import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { removeStorage } from './remove-storage.use-case'
import type { StorageNode } from './configured-outputs.use-case'

/** The command ids contributed in package.json. */
export const REMOVE_STORAGE_COMMAND = 'loreMaster.removeStorage'
/** Re-reads the Sync view, which lists the storages: after one is added, edited or removed. */
export const REFRESH_STORAGES_COMMAND = 'loreMaster.refreshPages'

/** Removes the storage the context menu was invoked on, after confirming. The node is the
 *  tree element VS Code passes from the view/item/context menu. */
export async function removeStorageCommand (deps: { engine: EngineClient }, node: StorageNode | undefined): Promise<void> {
  if (!node) {
    return
  }
  const folder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
  if (!folder) {
    return
  }

  const confirmed = await vscode.window.showWarningMessage('Remove this storage from your LoreMaster settings?', { modal: true }, 'Remove')
  if (confirmed !== 'Remove') {
    return
  }

  try {
    await removeStorage({ engine: deps.engine, workspaceRoot: folder, index: node.index })
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}
