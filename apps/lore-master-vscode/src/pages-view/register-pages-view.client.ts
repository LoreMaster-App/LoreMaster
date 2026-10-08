import * as vscode from 'vscode'
import type { SyncCommandDeps } from '../sync-command'
import { CHECK_REMOTE_COMMAND, checkRemoteCommand } from './check-remote.handler'
import { labelModeFrom } from './page-label.policy'
import { PAGES_VIEW_ID, type PagesNode, PagesViewProvider } from './pages-view.client'
import { SYNC_PAGE_COMMAND, syncPage } from './sync-page.handler'
import { PAGE_LABEL_SETTING, TOGGLE_PAGE_LABEL_COMMAND, togglePageLabel } from './toggle-page-label.handler'

/** The command id contributed in package.json. */
export const REFRESH_PAGES_COMMAND = 'loreMaster.refreshPages'

/** How long to wait for a burst of file changes to settle before re-reading the tree. */
const REFRESH_DEBOUNCE_MS = 300

/**
 * Registers the Pages view, its commands, and what keeps it current: a watcher on the
 * Markdown files, the settings file and the .gitignore files (a sync writes annotations
 * back into files, so it refreshes the view too), and the label setting.
 */
export function registerPagesView (deps: SyncCommandDeps): vscode.Disposable {
  const provider = new PagesViewProvider(
    deps.engine,
    () => labelModeFrom(vscode.workspace.getConfiguration('loreMaster').get(PAGE_LABEL_SETTING)),
  )

  let timer: ReturnType<typeof setTimeout> | undefined
  const refreshSoon = (): void => {
    clearTimeout(timer)
    timer = setTimeout(() => provider.refresh(), REFRESH_DEBOUNCE_MS)
  }
  const watchers = ['**/*.md', '**/.lore-master.yaml', '**/.gitignore'].map(pattern => {
    const watcher = vscode.workspace.createFileSystemWatcher(pattern)

    return vscode.Disposable.from(watcher.onDidCreate(refreshSoon), watcher.onDidChange(refreshSoon), watcher.onDidDelete(refreshSoon), watcher)
  })

  return vscode.Disposable.from(
    vscode.window.registerTreeDataProvider(PAGES_VIEW_ID, provider),
    vscode.commands.registerCommand(REFRESH_PAGES_COMMAND, () => provider.refresh()),
    vscode.commands.registerCommand(TOGGLE_PAGE_LABEL_COMMAND, () => togglePageLabel()),
    vscode.commands.registerCommand(CHECK_REMOTE_COMMAND, () => checkRemoteCommand(deps, provider)),
    vscode.commands.registerCommand(SYNC_PAGE_COMMAND, (node: PagesNode | undefined) => syncPage(deps, provider, node)),
    vscode.workspace.onDidChangeConfiguration(event => {
      if (event.affectsConfiguration(`loreMaster.${PAGE_LABEL_SETTING}`)) {
        provider.redraw()
      }
    }),
    vscode.workspace.onDidChangeWorkspaceFolders(() => provider.refresh()),
    ...watchers,
    { dispose: () => clearTimeout(timer) },
  )
}
