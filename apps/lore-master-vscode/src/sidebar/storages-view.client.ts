import * as vscode from 'vscode'
import { type ConfiguredOutput, readConfiguredOutputs } from './configured-outputs.use-case'

/** The view id contributed in package.json (inside the LoreMaster view container). */
export const STORAGES_VIEW_ID = 'loreMaster.storages'

/** One configured output, with its index in the settings' outputs list. */
export type StorageNode = ConfiguredOutput

/** The minimal engine surface the view needs. */
export interface StoragesEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/**
 * The LoreMaster "Storages" sidebar view: one row per configured output, read live from the
 * settings. It is how you see where a workspace syncs without opening them. Names, icons and
 * the second line come from the engine.
 */
export class StoragesViewProvider implements vscode.TreeDataProvider<StorageNode> {
  private readonly changed = new vscode.EventEmitter<void>()
  readonly onDidChangeTreeData = this.changed.event

  constructor (private readonly engine: StoragesEngine) {}

  /** Re-reads the outputs; call after one is added or removed. */
  refresh (): void {
    this.changed.fire()
  }

  async getChildren (node?: StorageNode): Promise<StorageNode[]> {
    if (node) {
      return []
    }
    const folder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
    if (!folder) {
      return []
    }
    try {
      const read = await readConfiguredOutputs(this.engine, folder)

      return read.outputs
    } catch {
      return []
    }
  }

  getTreeItem (node: StorageNode): vscode.TreeItem {
    const item = new vscode.TreeItem(node.label, vscode.TreeItemCollapsibleState.None)
    item.description = node.detail
    item.iconPath = new vscode.ThemeIcon(node.icon)
    item.contextValue = 'loreMasterStorage'

    return item
  }
}
