import * as vscode from 'vscode'
import {
  CHOOSE_VIEW_MODE_COMMAND,
  chooseViewMode,
  excludedModeFrom,
  PAGES_EXCLUDED_SETTING,
  PAGES_VIEW_SETTING,
  toggleExcluded,
  TOGGLE_EXCLUDED_COMMAND,
  viewModeFrom,
} from '../pages-display'
import type { SyncCommandDeps } from '../sync-command'
import {
  EXCLUDE_FROM_SYNC_COMMAND,
  excludeFromSyncCommand,
  INCLUDE_IN_SYNC_COMMAND,
  includeInSyncCommand,
  type SelectionTarget,
} from '../sync-selection'
import { CHECK_REMOTE_COMMAND, checkRemoteCommand } from './check-remote.handler'
import { labelModeFrom } from './page-label.policy'
import { PAGES_VIEW_ID, type PagesNode, PagesViewProvider } from './pages-view.client'
import { SYNC_PAGE_COMMAND, syncPage } from './sync-page.handler'
import { PAGE_LABEL_SETTING, TOGGLE_PAGE_LABEL_COMMAND, togglePageLabel } from './toggle-page-label.handler'

/** The command id contributed in package.json. */
export const REFRESH_PAGES_COMMAND = 'loreMaster.refreshPages'

/** How long to wait for a burst of file changes to settle before re-reading the tree. */
const REFRESH_DEBOUNCE_MS = 300

/** The file or folder a Pages view row stands for, or undefined for rows that are neither. */
function targetOf (node: PagesNode | undefined): SelectionTarget | undefined {
  switch (node?.kind) {
    case 'page': { return { path: node.entry.node.path, isFolder: false, index: node.index }
    }
    case 'left-out-file': { return { path: node.file.path, isFolder: false, index: node.index }
    }
    case 'folder': { return { path: node.folder.path, isFolder: true, index: node.index }
    }
    default: { return undefined
    }
  }
}

/**
 * Registers the Pages view, its commands, and what keeps it current: a watcher on the
 * Markdown files, the settings file and the .gitignore files (a sync writes annotations
 * back into files, so it refreshes the view too), and the display settings.
 */
export function registerPagesView (deps: SyncCommandDeps): vscode.Disposable {
  const configuration = (): vscode.WorkspaceConfiguration => vscode.workspace.getConfiguration('loreMaster')
  const provider = new PagesViewProvider(deps.engine, () => ({
    label:    labelModeFrom(configuration().get(PAGE_LABEL_SETTING)),
    view:     viewModeFrom(configuration().get(PAGES_VIEW_SETTING)),
    excluded: excludedModeFrom(configuration().get(PAGES_EXCLUDED_SETTING)),
  }))

  const selectionDeps = async (): Promise<Parameters<typeof excludeFromSyncCommand>[0] | undefined> => {
    const { folder, outputs } = await provider.configuredOutputs()
    if (folder === '') {
      return undefined
    }

    return { engine: deps.engine, workspaceRoot: folder, outputs: () => Promise.resolve(outputs), changed: () => provider.refresh() }
  }

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
    vscode.commands.registerCommand(CHOOSE_VIEW_MODE_COMMAND, () => chooseViewMode()),
    vscode.commands.registerCommand(TOGGLE_EXCLUDED_COMMAND, () => toggleExcluded()),
    vscode.commands.registerCommand(CHECK_REMOTE_COMMAND, () => checkRemoteCommand(deps, provider)),
    vscode.commands.registerCommand(SYNC_PAGE_COMMAND, (node: PagesNode | undefined) => syncPage(deps, provider, node)),
    vscode.commands.registerCommand(EXCLUDE_FROM_SYNC_COMMAND, async (node: PagesNode | undefined) => {
      const selection = await selectionDeps()
      if (selection) {
        await excludeFromSyncCommand(selection, targetOf(node))
      }
    }),
    vscode.commands.registerCommand(INCLUDE_IN_SYNC_COMMAND, async (node: PagesNode | undefined) => {
      const selection = await selectionDeps()
      if (selection) {
        await includeInSyncCommand(selection, targetOf(node))
      }
    }),
    vscode.workspace.onDidChangeConfiguration(event => {
      const names = [PAGE_LABEL_SETTING, PAGES_VIEW_SETTING, PAGES_EXCLUDED_SETTING]
      if (names.some(name => event.affectsConfiguration(`loreMaster.${name}`))) {
        provider.redraw()
      }
    }),
    vscode.workspace.onDidChangeWorkspaceFolders(() => provider.refresh()),
    ...watchers,
    { dispose: () => clearTimeout(timer) },
  )
}
