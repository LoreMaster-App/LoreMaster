import * as vscode from 'vscode'
import { SYNC_COMMAND } from '../sync-command'
import { ADD_STORAGE_COMMAND } from './add-storage.handler'
import { OPEN_CONFIG_COMMAND } from './open-config.handler'

/** The view id contributed in package.json (inside the LoreMaster view container). */
export const SYNC_VIEW_ID = 'loreMaster.sync'

/** One row in the Sync view: a label that runs a command. */
interface SyncAction {
  label:   string
  command: string
  icon:    string
}

const SYNC_ACTIONS: SyncAction[] = [
  { label: 'Add sync storage', command: ADD_STORAGE_COMMAND, icon: 'add' },
  { label: 'Sync', command: SYNC_COMMAND, icon: 'sync' },
  { label: 'Update config…', command: OPEN_CONFIG_COMMAND, icon: 'gear' },
]

/** The LoreMaster "Sync" sidebar view: a flat list of actions, each a single click. */
export class SyncViewProvider implements vscode.TreeDataProvider<SyncAction> {
  getTreeItem (action: SyncAction): vscode.TreeItem {
    const item = new vscode.TreeItem(action.label, vscode.TreeItemCollapsibleState.None)
    item.command = { command: action.command, title: action.label }
    item.iconPath = new vscode.ThemeIcon(action.icon)

    return item
  }

  getChildren (action?: SyncAction): SyncAction[] {
    return action ? [] : SYNC_ACTIONS
  }
}

/** Registers the Sync view; the returned disposable unregisters it on deactivate. */
export function registerSyncView (): vscode.Disposable {
  return vscode.window.registerTreeDataProvider(SYNC_VIEW_ID, new SyncViewProvider())
}
