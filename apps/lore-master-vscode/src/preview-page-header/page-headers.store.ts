import { relative, sep } from 'node:path'
import {
  SETTINGS_READ_METHOD, WORKSPACE_TREE_METHOD,
  type SettingsReadResult, type WorkspaceTreeResult,
} from '../engine-protocol'
import type { HeaderEngine, PageHeader, PageHeaderLookup } from './page-header.contract'

/** How long what the engine said about a workspace is trusted before the next render asks again. */
export const STALE_AFTER_MS = 3000

interface Workspace {
  loadedAt: number
  loading:  boolean
  files:    Map<string, PageHeader[]>
}

const keyOf = (root: string, documentPath: string): string => relative(root, documentPath).split(sep).join('/')

/**
 * What the engine says about each file of a workspace: the page title and status per storage. Markdown-it renders synchronously
 * and the engine answers later, so a file not yet known has no header, the engine is asked once, and `onChange` fires when the
 * answer differs from what was known so the preview renders again. A render after `STALE_AFTER_MS` asks again, so the status
 * follows edits and syncs.
 */
export class PageHeaders implements PageHeaderLookup {
  private readonly workspaces = new Map<string, Workspace>()

  constructor (
    private readonly engine: HeaderEngine,
    private readonly rootOf: (documentPath: string) => string | undefined,
    private readonly onChange: () => void,
    private readonly now: () => number = Date.now,
  ) {}

  private async load (root: string, workspace: Workspace): Promise<void> {
    workspace.loading = true
    const before = JSON.stringify([...workspace.files])
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
    } catch {
      // The engine is not ready or the settings are not valid: keep what was known and ask again after the next stale period.
    }
    workspace.loadedAt = this.now()
    workspace.loading = false
    if (JSON.stringify([...workspace.files]) !== before) { this.onChange() }
  }

  headersOf (documentPath: string): PageHeader[] | undefined {
    const root = this.rootOf(documentPath)
    if (root === undefined) { return undefined }
    let workspace = this.workspaces.get(root)
    if (!workspace) {
      workspace = { loadedAt: -Infinity, loading: false, files: new Map() }
      this.workspaces.set(root, workspace)
    }
    if (!workspace.loading && this.now() - workspace.loadedAt > STALE_AFTER_MS) { void this.load(root, workspace) }

    return workspace.files.get(keyOf(root, documentPath))
  }
}
