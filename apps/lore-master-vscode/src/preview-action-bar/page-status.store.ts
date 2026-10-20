import { relative, sep } from 'node:path'
import {
  SETTINGS_READ_METHOD, WORKSPACE_TREE_METHOD,
  type SettingsReadResult, type WorkspaceTreeResult,
} from '../engine-protocol'
import type { PageHeader, PageStatus, PageStatusLookup, StatusEngine } from './action-bar.contract'

/** How long what the engine said about a workspace is trusted before the next render asks again. */
export const STALE_AFTER_MS = 3000

interface Workspace {
  loadedAt: number
  loading:  boolean
  /** Whether an answer, or a failure, has come in yet. */
  settled:  boolean
  error?:   string
  files:    Map<string, PageHeader[]>
}

const keyOf = (root: string, documentPath: string): string => relative(root, documentPath).split(sep).join('/')

const messageOf = (error: unknown): string => error instanceof Error ? error.message : String(error)

const fingerprint = (workspace: Workspace): string => JSON.stringify({ files: [...workspace.files], error: workspace.error })

/**
 * What the engine says about each file of a workspace: the page title and status per storage. Markdown-it renders synchronously
 * and the engine answers later, so a file not yet known is `checking`, the engine is asked once, and `onChange` fires when the
 * answer differs from what was known so the preview renders again. A render after `STALE_AFTER_MS` asks again, so the status
 * follows edits and syncs. A failure is kept as the reason, and shown, rather than leaving the bar silent.
 */
export class PageStatuses implements PageStatusLookup {
  private readonly workspaces = new Map<string, Workspace>()

  constructor (
    private readonly engine: StatusEngine,
    private readonly rootOf: (documentPath: string) => string | undefined,
    private readonly onChange: () => void,
    private readonly now: () => number = Date.now,
  ) {}

  private async load (root: string, workspace: Workspace): Promise<void> {
    workspace.loading = true
    const wasSettled = workspace.settled
    const before = fingerprint(workspace)
    try {
      const settings = await this.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: root })
      const files = new Map<string, PageHeader[]>()
      for (const index of settings.settings.outputs.keys()) {
        const tree = await this.engine.request<WorkspaceTreeResult>(WORKSPACE_TREE_METHOD, { workspaceRoot: root, output: index })
        for (const node of tree.nodes) {
          files.set(node.path, [...files.get(node.path) ?? [], { storage: settings.displayNames[index], pageTitle: node.pageTitle, status: node.status }])
        }
      }
      workspace.files = files
      workspace.error = undefined
    } catch (error) {
      workspace.error = messageOf(error)
    }
    workspace.loadedAt = this.now()
    workspace.loading = false
    workspace.settled = true
    if (!wasSettled || fingerprint(workspace) !== before) { this.onChange() }
  }

  statusOf (documentPath: string): PageStatus {
    const root = this.rootOf(documentPath)
    if (root === undefined) { return { state: 'untracked' } }
    let workspace = this.workspaces.get(root)
    if (!workspace) {
      workspace = { loadedAt: -Infinity, loading: false, settled: false, files: new Map() }
      this.workspaces.set(root, workspace)
    }
    if (!workspace.loading && this.now() - workspace.loadedAt > STALE_AFTER_MS) { void this.load(root, workspace) }
    if (!workspace.settled) { return { state: 'checking' } }
    if (workspace.error !== undefined) { return { state: 'unavailable', reason: workspace.error } }
    const pages = workspace.files.get(keyOf(root, documentPath))

    return pages === undefined ? { state: 'untracked' } : { state: 'tracked', pages }
  }
}
