import { join } from 'node:path'
import * as vscode from 'vscode'
import {
  type LeftOutFile,
  type Output,
  SETTINGS_READ_METHOD,
  type SettingsReadResult,
  WORKSPACE_TREE_METHOD,
  type WorkspaceTreeResult,
} from '../engine-protocol'
import { type ExcludedMode, flatten, repoTree, type RepoFolder, type ViewMode } from '../pages-display'
import { isConfiguredOutput, storageDescription, storageIcon, storageLabel } from '../sidebar'
import { buildPageEntries } from './build-page-entries.algorithm'
import { leftOutReason } from './left-out-reason.policy'
import { pageDescription, pageLabel } from './page-label.policy'
import { statusPresentation } from './page-status.policy'
import type { LabelMode, PageEntry, RemoteCheck } from './page-tree.contract'

/** The view id contributed in package.json (inside the LoreMaster view container). */
export const PAGES_VIEW_ID = 'loreMaster.pages'

/** The index the Local node stands for: every Markdown file, whatever the storages leave out. */
export const LOCAL_INDEX = -1

/** The minimal engine surface the view needs. */
export interface PagesEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** One configured output, with its index in the settings' outputs list. */
export interface ConfiguredOutput {
  index:  number
  output: Output
}

/** How the view is arranged, read from the settings each time it is drawn. */
export interface DisplayOptions {
  label:    LabelMode
  view:     ViewMode
  excluded: ExcludedMode
}

/** A file in the repository tree: a page, or a file the storage leaves out. */
export type RepoItem =
  | { kind: 'page'; index: number; entry: PageEntry } |
  { kind: 'left-out-file'; index: number; file: LeftOutFile }

/** A row in the Pages view. */
export type PagesNode =
  | { kind: 'local' } |
  { kind: 'storage'; index: number; output: Output } |
  { kind: 'folder'; index: number; folder: RepoFolder<RepoItem> } |
  { kind: 'page'; index: number; entry: PageEntry } |
  { kind: 'orphan'; index: number; title: string; url?: string } |
  { kind: 'left-out'; index: number; files: LeftOutFile[]; total: number } |
  { kind: 'left-out-file'; index: number; file: LeftOutFile } |
  { kind: 'notice'; message: string }

const pathOfItem = (item: RepoItem): string => item.kind === 'page' ? item.entry.node.path : item.file.path

/**
 * The LoreMaster "Pages" sidebar view: every Markdown file as the tree it is, or will be, on
 * the storage, each with an icon for where it stands. A "Local" node shows every file of the
 * repository and which storages sync it; each storage node shows what that storage will, or
 * does, hold. The tree and the local half of the status come from the engine's `workspace/tree`
 * (no connection needed); the remote half is added by an explicit check and dropped again as
 * soon as a file changes, so the view never shows a platform state it has not just asked for.
 */
export class PagesViewProvider implements vscode.TreeDataProvider<PagesNode> {
  private readonly changed = new vscode.EventEmitter<PagesNode | undefined>()
  private folder = ''
  private trees = new Map<number, Promise<WorkspaceTreeResult>>()
  private remote = new Map<number, RemoteCheck>()
  private outputs: ConfiguredOutput[] = []

  readonly onDidChangeTreeData = this.changed.event

  constructor (private readonly engine: PagesEngine, private readonly options: () => DisplayOptions) {}

  private async pagesOf (index: number): Promise<PagesNode[]> {
    let tree: WorkspaceTreeResult
    try {
      tree = await this.treeOf(index)
    } catch (error) {
      return [{ kind: 'notice', message: error instanceof Error ? error.message : String(error) }]
    }

    const { label, view, excluded } = this.options()
    const local = index === LOCAL_INDEX
    const remote = local ? undefined : this.remote.get(index)
    const rows: PagesNode[] = (tree.problems ?? []).map(message => ({ kind: 'notice', message }))
    const entries = buildPageEntries(tree.nodes, label, remote)
    const leftOut = local || excluded === 'hidden' ? [] : tree.leftOut ?? []
    const leftOutTotal = local || excluded === 'hidden' ? 0 : tree.leftOutTotal ?? leftOut.length

    if (view === 'repo') {
      rows.push(...this.repoRows(index, entries, leftOut))
    } else {
      const shown = view === 'flat' ? flatten(allEntries(entries), each => each.node.path) : entries
      for (const entry of shown) {
        rows.push({ kind: 'page', index, entry: view === 'flat' ? { ...entry, children: [] } : entry })
      }
    }
    const orphans = remote?.orphans ?? []
    for (const orphan of orphans) {
      rows.push({ kind: 'orphan', index, title: orphan.title, url: orphan.url })
    }
    if (view !== 'repo' && leftOut.length > 0) {
      rows.push({ kind: 'left-out', index, files: leftOut, total: leftOutTotal })
    }

    return rows
  }

  /** The files placed where they sit in the repository, left-out ones in place and faded. */
  private repoRows (index: number, entries: PageEntry[], leftOut: LeftOutFile[]): PagesNode[] {
    const items: RepoItem[] = [
      ...allEntries(entries).map((entry): RepoItem => ({ kind: 'page', index, entry: { ...entry, children: [] } })),
      ...leftOut.map((file): RepoItem => ({ kind: 'left-out-file', index, file })),
    ]
    const root = repoTree(items, pathOfItem)

    return [...folderRows(index, root.folders), ...root.files.map(file => itemRow(file))]
  }

  private treeOf (index: number): Promise<WorkspaceTreeResult> {
    let tree = this.trees.get(index)
    if (!tree) {
      const params = index === LOCAL_INDEX
        ? { workspaceRoot: this.folder, output: 0, scope: 'local' }
        : { workspaceRoot: this.folder, output: index }
      tree = this.engine.request<WorkspaceTreeResult>(WORKSPACE_TREE_METHOD, params)
      this.trees.set(index, tree)
    }

    return tree
  }

  private localItem (): vscode.TreeItem {
    const item = new vscode.TreeItem('Local', vscode.TreeItemCollapsibleState.Collapsed)
    item.id = 'storage:local'
    item.description = 'every Markdown file in the repository'
    item.iconPath = new vscode.ThemeIcon('root-folder')
    item.tooltip = 'Every Markdown file of the workspace, including the ones a storage leaves out. Each row says which storages sync it.'
    item.contextValue = 'loreMasterPagesLocal'

    return item
  }

  private storageItem (node: { index: number; output: Output }): vscode.TreeItem {
    const { output } = node
    const item = new vscode.TreeItem(storageLabel(output), vscode.TreeItemCollapsibleState.Expanded)
    item.id = `storage:${node.index}`
    item.description = storageDescription(output)
    item.iconPath = new vscode.ThemeIcon(storageIcon(output))
    item.contextValue = 'loreMasterPagesStorage'

    return item
  }

  private folderItem (node: { index: number; folder: RepoFolder<RepoItem> }): vscode.TreeItem {
    const item = new vscode.TreeItem(node.folder.name, vscode.TreeItemCollapsibleState.Expanded)
    item.id = `folder:${node.index}:${node.folder.path}`
    item.iconPath = vscode.ThemeIcon.Folder
    item.resourceUri = vscode.Uri.file(join(this.folder, node.folder.path))
    item.contextValue = 'loreMasterFolder'

    return item
  }

  /** Where a local file is synced, in words: the storages by name, or why it is not. */
  private syncedTo (indexes: number[] | undefined, gitIgnored: boolean | undefined): string {
    if (gitIgnored) {
      return 'ignored by git, never synced'
    }
    const names = (indexes ?? []).map(index => this.outputs.find(each => each.index === index)).filter(each => each !== undefined)
      .map(each => storageLabel(each.output))

    return names.length > 0 ? `→ ${names.join(', ')}` : 'not synced'
  }

  private pageItem (node: { index: number; entry: PageEntry }): vscode.TreeItem {
    const { entry } = node
    const { label: mode } = this.options()
    const local = node.index === LOCAL_INDEX
    const item = new vscode.TreeItem(pageLabel(entry.node, mode), entry.children.length > 0 ? vscode.TreeItemCollapsibleState.Collapsed : vscode.TreeItemCollapsibleState.None)
    item.id = `page:${node.index}:${entry.node.path}`
    item.description = local
      ? `${pageDescription(entry.node, mode)} · ${this.syncedTo(entry.node.syncedTo, entry.node.gitIgnored)}`
      : pageDescription(entry.node, mode)
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
    } else if (local && entry.node.gitIgnored) {
      item.iconPath = new vscode.ThemeIcon('markdown', new vscode.ThemeColor('disabledForeground'))
      item.contextValue = 'loreMasterIgnoredFile'
      lines.push('', 'Git ignores this file, so no storage syncs it.')
    } else {
      item.iconPath = new vscode.ThemeIcon('markdown')
      item.contextValue = 'loreMasterPageUntracked'
    }
    lines.push(...(entry.node.warnings ?? []).map(warning => `! ${warning}`))
    item.tooltip = lines.join('\n')

    return item
  }

  private leftOutItem (node: { index: number; total: number }): vscode.TreeItem {
    const item = new vscode.TreeItem(`Left out (${node.total})`, vscode.TreeItemCollapsibleState.Collapsed)
    item.id = `left-out:${node.index}`
    item.description = 'Markdown the sync does not read'
    item.iconPath = new vscode.ThemeIcon('eye-closed')
    item.tooltip = 'Markdown files this storage leaves out, and the setting that leaves each one out.'
    item.contextValue = 'loreMasterLeftOut'

    return item
  }

  private leftOutFileItem (node: { index: number; file: LeftOutFile }, inTree: boolean): vscode.TreeItem {
    const reason = leftOutReason(node.file)
    const label = inTree ? node.file.path.slice(node.file.path.lastIndexOf('/') + 1) : node.file.path
    const item = new vscode.TreeItem(label, vscode.TreeItemCollapsibleState.None)
    item.id = `left-out-file:${node.index}:${node.file.path}`
    item.description = reason.description
    item.iconPath = new vscode.ThemeIcon('markdown', new vscode.ThemeColor('disabledForeground'))
    item.tooltip = `${node.file.path}
${reason.detail}`
    item.command = { command: 'vscode.open', title: 'Open', arguments: [vscode.Uri.file(join(this.folder, node.file.path))] }
    // A file git ignores, or that sits outside the storage's roots, cannot be brought back by an
    // include: the first is never synced, the second is not read at all.
    item.contextValue = node.file.rule === 'gitignore' || node.file.rule === 'outside-roots' ? 'loreMasterIgnoredFile' : 'loreMasterLeftOutFile'

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

  /** Only the naming or arrangement changed: redraw without losing the platform's answer. */
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
    if (node?.kind === 'local') {
      return this.pagesOf(LOCAL_INDEX)
    }
    if (node?.kind === 'storage') {
      return this.pagesOf(node.index)
    }
    if (node?.kind === 'folder') {
      return [...folderRows(node.index, node.folder.folders), ...node.folder.files.map(file => itemRow(file))]
    }
    if (node?.kind === 'left-out') {
      const rows: PagesNode[] = node.files.map(file => ({ kind: 'left-out-file', index: node.index, file }))
      if (node.total > node.files.length) {
        rows.push({ kind: 'notice', message: `… and ${node.total - node.files.length} more` })
      }

      return rows
    }
    if (node) {
      return []
    }

    const { folder, outputs } = await this.configuredOutputs()
    this.folder = folder
    this.outputs = outputs
    if (outputs.length === 0) {
      return []
    }

    return [{ kind: 'local' }, ...outputs.map((each): PagesNode => ({ kind: 'storage', index: each.index, output: each.output }))]
  }

  getTreeItem (node: PagesNode): vscode.TreeItem {
    switch (node.kind) {
      case 'local': { return this.localItem()
      }
      case 'storage': { return this.storageItem(node)
      }
      case 'folder': { return this.folderItem(node)
      }
      case 'page': { return this.pageItem(node)
      }
      case 'orphan': { return this.orphanItem(node)
      }
      case 'left-out': { return this.leftOutItem(node)
      }
      case 'left-out-file': { return this.leftOutFileItem(node, this.options().view === 'repo')
      }
      case 'notice': { return noticeItem(node)
      }
    }
  }
}

/** Every entry of the nested tree, parents before their children. */
function allEntries (entries: PageEntry[]): PageEntry[] {
  return entries.flatMap(entry => [entry, ...allEntries(entry.children)])
}

function itemRow (item: RepoItem): PagesNode {
  return item.kind === 'page'
    ? { kind: 'page', index: item.index, entry: item.entry }
    : { kind: 'left-out-file', index: item.index, file: item.file }
}

function folderRows (index: number, folders: RepoFolder<RepoItem>[]): PagesNode[] {
  return folders.map(folder => ({ kind: 'folder', index, folder }))
}

function noticeItem (node: { message: string }): vscode.TreeItem {
  const item = new vscode.TreeItem(node.message, vscode.TreeItemCollapsibleState.None)
  item.iconPath = new vscode.ThemeIcon('error', new vscode.ThemeColor('list.errorForeground'))
  item.tooltip = node.message

  return item
}
