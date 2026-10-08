import * as vscode from 'vscode'
import { type Output, SETTINGS_READ_METHOD, type SettingsReadResult } from '../engine-protocol'

/** The view id contributed in package.json (inside the LoreMaster view container). */
export const STORAGES_VIEW_ID = 'loreMaster.storages'

/** One configured output, with its index in the settings' outputs list. */
export interface StorageNode {
  index:  number
  output: Output
}

/** The minimal engine surface the view needs. */
export interface StoragesEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/**
 * The LoreMaster "Storages" sidebar view: one row per configured output, read live from
 * .lore-master.yaml. It is how you see where a workspace syncs without opening the YAML.
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
    let read: SettingsReadResult
    try {
      read = await this.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: folder })
    } catch {
      return []
    }

    return read.settings.outputs
      .map((output, index) => ({ index, output }))
      .filter(node => isConfiguredOutput(node.output))
  }

  getTreeItem (node: StorageNode): vscode.TreeItem {
    const item = new vscode.TreeItem(storageLabel(node.output), vscode.TreeItemCollapsibleState.None)
    item.description = storageDescription(node.output)
    item.iconPath = new vscode.ThemeIcon(node.output.platform === 'github-pages' ? 'globe' : 'book')
    item.contextValue = 'loreMasterStorage'

    return item
  }
}

/** An output shows in the list once it is actually configured (the blank scaffold is hidden). */
export function isConfiguredOutput (output: Output): boolean {
  if (output.platform === 'github-pages') {
    return true
  }

  return output.platform === 'confluence' && output.baseUrl !== '' && output.space !== ''
}

export function storageLabel (output: Output): string {
  return output.platform === 'github-pages' ? `GitHub Pages · ${output.branch || 'gh-pages'}` : `Confluence · ${output.space}`
}

export function storageDescription (output: Output): string {
  return output.platform === 'github-pages' ? (output.repo || "this repo's origin") : output.baseUrl
}
