import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { removeStorage } from './remove-storage.use-case'
import type { StorageNode } from './storages-view.client'

/** The command ids contributed in package.json. */
export const REMOVE_STORAGE_COMMAND = 'loreMaster.removeStorage'
export const REFRESH_STORAGES_COMMAND = 'loreMaster.refreshStorages'

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

  const confirmed = await vscode.window.showWarningMessage('Remove this storage from .lore-master.yaml?', { modal: true }, 'Remove')
  if (confirmed !== 'Remove') {
    return
  }

  try {
    const result = await removeStorage({ engine: deps.engine, workspaceRoot: folder, index: node.index })
    if (result === 'last') {
      await vscode.window.showInformationMessage('That is the only storage — add another first, or open .lore-master.yaml to remove it by hand.')
    }
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}
