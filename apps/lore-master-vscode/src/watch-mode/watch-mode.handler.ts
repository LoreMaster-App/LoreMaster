import { createHash } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import { join } from 'node:path'
import * as vscode from 'vscode'
import { SETTINGS_READ_METHOD, type SettingsReadResult } from '../engine-protocol'
import { syncOutputs, type SyncCommandDeps } from '../sync-command'
import { pickWorkspaceFolder } from '../workspace-files'
import { applyChanges } from './apply-changes.use-case'
import { ChangeBatcher } from './change-batching.use-case'
import { createSilentSyncUI } from './silent-sync-ui.client'
import { isWatchedPath, workspacePath } from './watched-paths.policy'

/** The command id contributed in package.json. */
export const TOGGLE_WATCH_COMMAND = 'loreMaster.toggleWatch'

/** The setting (in seconds) for how long nothing may change before the changes are synced. */
export const WATCH_DEBOUNCE_SETTING = 'watchDebounceSeconds'

const DEFAULT_DEBOUNCE_SECONDS = 2
const MAX_BACKOFF_MS = 60_000
const WATCHING = '$(eye) LoreMaster: watching'

/** What a running watch holds, so it can be stopped. */
interface Session {
  folder:       string
  watcher:      vscode.FileSystemWatcher
  batcher:      ChangeBatcher
  item:         vscode.StatusBarItem
  /** What the files our own batches wrote contained afterwards, to tell that from the user's edits. */
  fingerprints: Map<string, string>
}

/**
 * Watch mode: while it is on, a change to a Markdown file, the settings, or a file a generator
 * reads is synced to the storage on its own once nothing has changed for a moment. Only the
 * generators that read what changed run, and only the pages that changed are synced, without
 * asking anything; the status bar shows what it is doing, and the LoreMaster output channel logs it.
 * The command line's `watch` does the same.
 */
export class WatchMode implements vscode.Disposable {
  private session: Session | undefined

  constructor (private readonly deps: SyncCommandDeps) {}

  private async start (): Promise<void> {
    const folder = await pickWorkspaceFolder()
    if (!folder) {
      await vscode.window.showInformationMessage('LoreMaster: open a folder to watch it.')

      return
    }
    try {
      const read = await this.deps.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: folder })
      if (read.settings.outputs.length === 0) {
        await vscode.window.showInformationMessage('LoreMaster: set up a storage first (run Sync once); watching needs somewhere to send the pages.')

        return
      }
    } catch (error) {
      await vscode.window.showErrorMessage(`LoreMaster: ${messageOf(error)}`)

      return
    }

    const item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 0)
    item.command = TOGGLE_WATCH_COMMAND
    const seconds = vscode.workspace.getConfiguration('loreMaster').get<number>(WATCH_DEBOUNCE_SETTING, DEFAULT_DEBOUNCE_SECONDS)
    const quietMs = Math.max(1, seconds) * 1000
    const fingerprints = new Map<string, string>()
    const batcher = new ChangeBatcher({
      quietMs,
      maxBackoffMs: Math.max(MAX_BACKOFF_MS, quietMs),
      run:          changed => this.run(folder, item, fingerprints, changed),
      onFailure:    (error, retryInMs) => {
        this.log(`failed: ${messageOf(error)}; trying again in ${Math.round(retryInMs / 1000)}s`)
        item.text = '$(warning) LoreMaster: sync failed, retrying'
        item.tooltip = messageOf(error)
      },
    })

    const watcher = vscode.workspace.createFileSystemWatcher(new vscode.RelativePattern(folder, '**/*'))
    const onChange = (uri: vscode.Uri): void => { void this.noticeChange(folder, fingerprints, batcher, uri.fsPath) }
    watcher.onDidCreate(onChange)
    watcher.onDidChange(onChange)
    watcher.onDidDelete(onChange)

    this.session = { folder, watcher, batcher, item, fingerprints }
    this.showWatching(item, folder)
    item.show()
    this.log(`watching ${folder} (syncing after ${quietMs / 1000}s of quiet)`)
    await vscode.window.showInformationMessage(`LoreMaster is watching ${folder}: changes sync to the storage on their own. Run "Toggle Watch Mode" to stop.`)
  }

  private stop (): void {
    const session = this.session
    if (!session) {
      return
    }
    this.session = undefined
    session.batcher.dispose()
    session.watcher.dispose()
    session.item.dispose()
    this.log('stopped watching')
  }

  /** Queues a changed file, unless it only holds what a batch of ours just wrote there. */
  private async noticeChange (folder: string, fingerprints: Map<string, string>, batcher: ChangeBatcher, fsPath: string): Promise<void> {
    const path = workspacePath(folder, fsPath)
    if (path === undefined || !isWatchedPath(path)) {
      return
    }
    const known = fingerprints.get(path)
    if (known !== undefined) {
      if (await fingerprint(join(folder, path)) === known) {
        return
      }
      fingerprints.delete(path)
    }
    batcher.add([path])
  }

  private async run (folder: string, item: vscode.StatusBarItem, fingerprints: Map<string, string>, changed: string[]): Promise<void> {
    item.text = '$(sync~spin) LoreMaster: syncing'
    this.log(`changed: ${changed.join(', ')}`)
    const ui = createSilentSyncUI(this.deps.output)
    const applied = await applyChanges({
      engine:        this.deps.engine,
      workspaceRoot: folder,
      log:           line => { this.log(line) },
      sync:          async scope => {
        ui.problems.length = 0
        await syncOutputs({ engine: this.deps.engine, connections: this.deps.connections, targets: this.deps.targets, workspaceRoot: folder, scope, ui })
        if (ui.problems.length > 0) {
          throw new Error(ui.problems[0])
        }
      },
    }, changed)

    for (const path of applied.touched) {
      fingerprints.set(path, await fingerprint(join(folder, path)))
    }
    this.showWatching(item, folder)
    this.log(`done (${changed.length} file(s))`)
  }

  private showWatching (item: vscode.StatusBarItem, folder: string): void {
    item.text = WATCHING
    item.tooltip = `Watching ${folder}. Click to stop.`
  }

  private log (line: string): void {
    this.deps.output.appendLine(`[${new Date().toLocaleTimeString()}] ${line}`)
  }

  /** Turns watch mode on for the workspace folder, or off when it is on. */
  async toggle (): Promise<void> {
    if (this.session) {
      this.stop()
      await vscode.window.showInformationMessage('LoreMaster: stopped watching.')

      return
    }
    await this.start()
  }

  dispose (): void {
    this.stop()
  }

  get watching (): boolean {
    return this.session !== undefined
  }
}

/** A hash of a file's content, or a marker when it is not there. */
async function fingerprint (path: string): Promise<string> {
  try {
    return createHash('sha256').update(await readFile(path)).digest('hex')
  } catch {
    return 'missing'
  }
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
