import { join } from 'node:path'
import * as vscode from 'vscode'
import {
  type Output,
  SETTINGS_READ_METHOD,
  type SettingsReadResult,
  WORKSPACE_TREE_METHOD,
  type WorkspaceTreeResult,
} from '../engine-protocol'
import { isConfiguredOutput } from '../sidebar'
import { buildPageEntries } from './build-page-entries.algorithm'
import { pageDescription, pageLabel } from './page-label.policy'
import { statusPresentation } from './page-status.policy'
import type { LabelMode, PageEntry, RemoteCheck } from './page-tree.contract'

/** The view id contributed in package.json (inside the LoreMaster view container). */
export const PAGES_VIEW_ID = 'loreMaster.pages'

/** The minimal engine surface the view needs. */
export interface PagesEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** One configured output, with its index in the settings' outputs list. */
export interface ConfiguredOutput {
  index:  number
  output: Output
}

/** A row in the Pages view. */
export type PagesNode =
  | { kind: 'storage'; index: number; output: Output } |
  { kind: 'page'; index: number; entry: PageEntry } |
  { kind: 'orphan'; index: number; title: string; url?: string } |
  { kind: 'notice'; message: string }

/**
 * The LoreMaster "Pages" sidebar view: every Markdown file as the tree it is, or will be, on
 * the storage, each with an icon for where it stands. The tree and the local half of the
 * status come from the engine's `workspace/tree` (no connection needed); the remote half is
 * added by an explicit check and dropped again as soon as a file changes, so the view never
 * shows a platform state it has not just asked for.
 */
export class PagesViewProvider implements vscode.TreeDataProvider<PagesNode> {
  private readonly changed = new vscode.EventEmitter<PagesNode | undefined>()
  private folder = ''
  private trees = new Map<number, Promise<WorkspaceTreeResult>>()
  private remote = new Map<number, RemoteCheck>()

  readonly onDidChangeTreeData = this.changed.event

  constructor (private readonly engine: PagesEngine, private readonly labelMode: () => LabelMode) {}

  private async pagesOf (index: number): Promise<PagesNode[]> {
    let tree: WorkspaceTreeResult
    try {
      tree = await this.treeOf(index)
    } catch (error) {
      return [{ kind: 'notice', message: error instanceof Error ? error.message : String(error) }]
    }

    const remote = this.remote.get(index)
    const rows: PagesNode[] = (tree.problems ?? []).map(message => ({ kind: 'notice', message }))
    const entries = buildPageEntries(tree.nodes, this.labelMode(), remote)
    for (const entry of entries) {
      rows.push({ kind: 'page', index, entry })
    }
    const orphans = remote?.orphans ?? []
    for (const orphan of orphans) {
      rows.push({ kind: 'orphan', index, title: orphan.title, url: orphan.url })
    }

    return rows
  }

  private treeOf (index: number): Promise<WorkspaceTreeResult> {
    let tree = this.trees.get(index)
    if (!tree) {
      tree = this.engine.request<WorkspaceTreeResult>(WORKSPACE_TREE_METHOD, { workspaceRoot: this.folder, output: index })
      this.trees.set(index, tree)
    }

    return tree
  }

  private storageItem (node: { index: number; output: Output }): vscode.TreeItem {
    const { output } = node
    const item = new vscode.TreeItem(output.platform === 'github-pages' ? `GitHub Pages · ${output.branch || 'gh-pages'}` : `Confluence · ${output.space}`, vscode.TreeItemCollapsibleState.Expanded)
    item.id = `storage:${node.index}`
    item.description = output.platform === 'github-pages' ? (output.repo || "this repo's origin") : output.baseUrl
    item.iconPath = new vscode.ThemeIcon(output.platform === 'github-pages' ? 'globe' : 'book')
    item.contextValue = 'loreMasterPagesStorage'

    return item
  }

  private pageItem (node: { index: number; entry: PageEntry }): vscode.TreeItem {
    const { entry } = node
    const mode = this.labelMode()
    const item = new vscode.TreeItem(pageLabel(entry.node, mode), entry.children.length > 0 ? vscode.TreeItemCollapsibleState.Collapsed : vscode.TreeItemCollapsibleState.None)
    item.id = `page:${node.index}:${entry.node.path}`
    item.description = pageDescription(entry.node, mode)
    item.command = { command: 'vscode.open', title: 'Open', arguments: [vscode.Uri.file(join(this.folder, entry.node.path))] }

    const lines = [entry.node.pageTitle, entry.node.path]
    if (entry.status) {
      const presentation = statusPresentation(entry.status)
      item.iconPath = new vscode.ThemeIcon(presentation.icon, new vscode.ThemeColor(presentation.color))
      item.contextValue = 'loreMasterPage'
      lines.push('', `${presentation.label}: ${presentation.detail}`)
      if (!entry.remoteChecked) {
        lines.push('The platform has not been checked; run "Check remote status".')
      }
    } else {
      item.iconPath = new vscode.ThemeIcon('markdown')
      item.contextValue = 'loreMasterPageUntracked'
    }
    lines.push(...(entry.node.warnings ?? []).map(warning => `! ${warning}`))
    item.tooltip = lines.join('\n')

    return item
  }

  private orphanItem (node: { title: string; url?: string }): vscode.TreeItem {
    const item = new vscode.TreeItem(node.title, vscode.TreeItemCollapsibleState.None)
    item.description = 'no file any more'
    item.iconPath = new vscode.ThemeIcon('trash', new vscode.ThemeColor('list.warningForeground'))
    item.tooltip = `${node.title}\nA page the sync made whose file is gone; prune moves it to the trash.${node.url ? `\n${node.url}` : ''}`
    item.contextValue = 'loreMasterOrphan'

    return item
  }

  /** The files changed: re-read the trees and forget what the platform said. */
  refresh (): void {
    this.trees = new Map()
    this.remote = new Map()
    this.changed.fire(undefined)
  }

  /** Only the naming changed (title or file name): redraw without losing the platform's answer. */
  redraw (): void {
    this.changed.fire(undefined)
  }

  /** Keeps what the platform said about an output until the files next change. */
  setRemote (index: number, check: RemoteCheck): void {
    this.remote.set(index, check)
    this.changed.fire(undefined)
  }

  /** The configured outputs of the first workspace folder, and the folder itself. */
  async configuredOutputs (): Promise<{ folder: string; outputs: ConfiguredOutput[] }> {
    const folder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
    if (!folder) {
      return { folder: '', outputs: [] }
    }
    try {
      const read = await this.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: folder })

      return {
        folder,
        outputs: read.settings.outputs.map((output, index) => ({ index, output })).filter(each => isConfiguredOutput(each.output)),
      }
    } catch {
      return { folder, outputs: [] }
    }
  }

  async getChildren (node?: PagesNode): Promise<PagesNode[]> {
    if (node?.kind === 'page') {
      return node.entry.children.map(entry => ({ kind: 'page', index: node.index, entry }))
    }
    if (node?.kind === 'storage') {
      return this.pagesOf(node.index)
    }
    if (node) {
      return []
    }

    const { folder, outputs } = await this.configuredOutputs()
    this.folder = folder
    if (outputs.length === 1) {
      return this.pagesOf(outputs[0].index)
    }

    return outputs.map(each => ({ kind: 'storage', index: each.index, output: each.output }))
  }

  getTreeItem (node: PagesNode): vscode.TreeItem {
    switch (node.kind) {
      case 'storage': { return this.storageItem(node)
      }
      case 'page': { return this.pageItem(node)
      }
      case 'orphan': { return this.orphanItem(node)
      }
      case 'notice': { return noticeItem(node)
      }
    }
  }
}

function noticeItem (node: { message: string }): vscode.TreeItem {
  const item = new vscode.TreeItem(node.message, vscode.TreeItemCollapsibleState.None)
  item.iconPath = new vscode.ThemeIcon('error', new vscode.ThemeColor('list.errorForeground'))
  item.tooltip = node.message

  return item
}
