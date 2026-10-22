import { join, relative } from 'node:path'
import * as vscode from 'vscode'
import {
  type ConfigDescription,
  type LeftOutFile,
  WORKSPACE_TREE_METHOD,
  type WorkspaceTreeResult,
} from '../engine-protocol'
import { flatten, repoTree, type RepoFolder } from '../pages-display'
import { type ConfiguredOutput, readConfiguredOutputs } from '../sidebar'
import { buildPageEntries } from './build-page-entries.algorithm'
import { leftOutReason } from './left-out-reason.policy'
import { pageDescription, pageLabel } from './page-label.policy'
import { statusPresentation } from './page-status.policy'
import type { DisplayOptions, PageEntry, RemoteCheck } from './page-tree.contract'
import { rowContext } from './row-context.policy'
import { rowState, rowStateLines } from './row-state.policy'

/** The view id contributed in package.json (inside the LoreMaster view container). */
export const PAGES_VIEW_ID = 'loreMaster.sync'

/** The index the Local node stands for: every Markdown file, whatever the storages leave out. */
export const LOCAL_INDEX = -1

/** The minimal engine surface the view needs. */
export interface PagesEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** A file in the repository tree: a page, or a file the storage leaves out. */
export type RepoItem =
  | { kind: 'page'; index: number; entry: PageEntry } |
  { kind: 'left-out-file'; index: number; file: LeftOutFile }

/** A row in the Sync view. */
export type PagesNode =
  | { kind: 'local' } |
  ({ kind: 'storage' } & ConfiguredOutput) |
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

  /** `options` says how one location is shown: Local is index -1, a storage its index in the settings. */
  constructor (private readonly engine: PagesEngine, private readonly options: (index: number) => DisplayOptions) {}

  /** What a location lists: its pages as nested entries, and the files it leaves out. */
  private shapeOf (tree: WorkspaceTreeResult, index: number): { entries: PageEntry[]; leftOut: LeftOutFile[]; leftOutTotal: number; remote: RemoteCheck | undefined } {
    const { label, excluded } = this.options(index)
    const local = index === LOCAL_INDEX
    const remote = local ? undefined : this.remote.get(index)
    const leftOut = local || excluded === 'hidden' ? [] : tree.leftOut ?? []

    // Local lists every file, so what it can hide is the Markdown git ignores; a storage hides what it leaves out.
    const listed = local && excluded === 'hidden' ? tree.nodes.filter(node => node.gitIgnored !== true) : tree.nodes

    return {
      entries:      buildPageEntries(listed, label, remote),
      leftOut,
      leftOutTotal: local || excluded === 'hidden' ? 0 : tree.leftOutTotal ?? leftOut.length,
      remote,
    }
  }

  private async pagesOf (index: number): Promise<PagesNode[]> {
    let tree: WorkspaceTreeResult
    try {
      tree = await this.treeOf(index)
    } catch (error) {
      return [{ kind: 'notice', message: error instanceof Error ? error.message : String(error) }]
    }

    const { view } = this.options(index)
    const rows: PagesNode[] = (tree.problems ?? []).map(message => ({ kind: 'notice', message }))
    const { entries, leftOut, leftOutTotal, remote } = this.shapeOf(tree, index)

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
    const root = this.repoRoot(index, entries, leftOut)

    return [...folderRows(index, root.folders), ...root.files.map(file => itemRow(file))]
  }

  private repoRoot (index: number, entries: PageEntry[], leftOut: LeftOutFile[]): RepoFolder<RepoItem> {
    const items: RepoItem[] = [
      ...allEntries(entries).map((entry): RepoItem => ({ kind: 'page', index, entry: { ...entry, children: [] } })),
      ...leftOut.map((file): RepoItem => ({ kind: 'left-out-file', index, file })),
    ]

    return repoTree(items, pathOfItem)
  }

  /** The row of a location: Local, or a storage. */
  private locationNode (index: number): PagesNode | undefined {
    if (index === LOCAL_INDEX) { return { kind: 'local' } }
    const output = this.outputs.find(each => each.index === index)

    return output === undefined ? undefined : { kind: 'storage', ...output }
  }

  /** The rows from a location's top level down to the one at `path` (a page or file, or a folder), as the location shows it now. */
  private async rowsTo (index: number, path: string, isFolder: boolean): Promise<PagesNode[]> {
    let tree: WorkspaceTreeResult
    try {
      tree = await this.treeOf(index)
    } catch {
      return []
    }
    const { view } = this.options(index)
    const { entries, leftOut } = this.shapeOf(tree, index)
    if (view === 'repo') {
      const chain: PagesNode[] = []
      let folder = this.repoRoot(index, entries, leftOut)
      while (true) {
        const next = folder.folders.find(each => path.startsWith(`${each.path}/`) || path === each.path)
        if (next === undefined) { break }
        chain.push({ kind: 'folder', index, folder: next })
        if (isFolder && next.path === path) { return chain }
        folder = next
      }
      const file = folder.files.find(each => pathOfItem(each) === path)

      return file === undefined || isFolder ? [] : [...chain, itemRow(file)]
    }
    if (isFolder) { return [] }
    if (view === 'flat') {
      const found = allEntries(entries).find(each => each.node.path === path)

      return found === undefined ? [] : [{ kind: 'page', index, entry: { ...found, children: [] } }]
    }

    return pageChain(index, entries, path)
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
    const options = this.options(LOCAL_INDEX)
    item.description = rowState(options)
    item.iconPath = new vscode.ThemeIcon('root-folder')
    item.tooltip = ['Every Markdown file of the workspace, including the ones a storage leaves out. Each row says which storages sync it.', '', ...rowStateLines(options, 'Files git ignores')].join('\n')
    item.contextValue = rowContext('loreMasterLocal', this.options(LOCAL_INDEX))

    return item
  }

  private storageItem (node: ConfiguredOutput): vscode.TreeItem {
    const item = new vscode.TreeItem(node.label, vscode.TreeItemCollapsibleState.Expanded)
    item.id = `storage:${node.index}`
    const options = this.options(node.index)
    item.description = node.detail === '' ? rowState(options) : `${rowState(options)} · ${node.detail}`
    item.iconPath = new vscode.ThemeIcon(node.icon)
    item.tooltip = [node.detail, '', ...rowStateLines(options, 'Files this storage leaves out')].join('\n')
    item.contextValue = rowContext('loreMasterStorage', this.options(node.index))

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
      .map(each => each.label)

    return names.length > 0 ? `→ ${names.join(', ')}` : 'not synced'
  }

  private pageItem (node: { index: number; entry: PageEntry }): vscode.TreeItem {
    const { entry } = node
    const { label: mode } = this.options(node.index)
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

  /** One location was asked to refresh: re-read its tree and forget what the platform said about it. */
  refreshLocation (index: number): void {
    this.trees.delete(index)
    this.remote.delete(index)
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
  async configuredOutputs (): Promise<{ folder: string; outputs: ConfiguredOutput[]; description?: ConfigDescription }> {
    const folder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
    if (!folder) {
      return { folder: '', outputs: [] }
    }
    try {
      return { folder, ...await readConfiguredOutputs(this.engine, folder) }
    } catch {
      return { folder, outputs: [] }
    }
  }

  /** The row for a Markdown file of the workspace in the Local location, or undefined when the view does not list it. */
  async locateFile (fsPath: string): Promise<PagesNode | undefined> {
    const path = relative(this.folder, fsPath).replaceAll('\\', '/')
    if (path === '' || this.folder === '' || path.startsWith('..')) { return undefined }
    const rows = await this.rowsTo(LOCAL_INDEX, path, false)

    return rows.at(-1)
  }

  async getParent (node: PagesNode): Promise<PagesNode | undefined> {
    if (!hasLocation(node)) { return undefined }
    const chain = await this.rowsTo(node.index, pathOfRow(node) ?? '', node.kind === 'folder')

    return chain.at(-2) ?? this.locationNode(node.index)
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

    return [{ kind: 'local' }, ...outputs.map((each): PagesNode => ({ kind: 'storage', ...each }))]
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
      case 'left-out-file': { return this.leftOutFileItem(node, this.options(node.index).view === 'repo')
      }
      case 'notice': { return noticeItem(node)
      }
    }
  }
}

/** Whether a row sits inside a location (Local or a storage), rather than being one or a bare notice. */
function hasLocation (node: PagesNode): node is Exclude<PagesNode, { kind: 'local' | 'storage' | 'notice' }> {
  return !['local', 'storage', 'notice'].includes(node.kind)
}

/** The workspace-relative path a row stands for, when it stands for one. */
function pathOfRow (node: PagesNode): string | undefined {
  switch (node.kind) {
    case 'page': { return node.entry.node.path
    }
    case 'folder': { return node.folder.path
    }
    case 'left-out-file': { return node.file.path
    }
    default: { return undefined
    }
  }
}

/** Every entry of the nested tree, parents before their children. */
function allEntries (entries: PageEntry[]): PageEntry[] {
  return entries.flatMap(entry => [entry, ...allEntries(entry.children)])
}

/** The page rows from the top of the tree down to the entry at `path`; empty when it is not there. */
function pageChain (index: number, entries: PageEntry[], path: string): PagesNode[] {
  for (const entry of entries) {
    const row: PagesNode = { kind: 'page', index, entry }
    if (entry.node.path === path) { return [row] }
    const below = pageChain(index, entry.children, path)
    if (below.length > 0) { return [row, ...below] }
  }

  return []
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
