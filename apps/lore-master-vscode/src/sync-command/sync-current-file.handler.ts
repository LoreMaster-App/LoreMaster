import { relative } from 'node:path'
import * as vscode from 'vscode'
import { connectConfluence } from '../connection-setup'
import { folderForActiveEditor } from '../workspace-files'
import type { SyncCommandDeps } from './sync.handler'
import { syncOutputs } from './sync-outputs.use-case'
import { createSyncUI } from './sync-ui.client'

/** The command id contributed in package.json. */
export const SYNC_CURRENT_FILE_COMMAND = 'loreMaster.syncCurrentFile'

/** Syncs just the active file (and the ancestors it needs) to the Confluence outputs,
 *  scoping the plan to it. GitHub Pages publishes the whole site, so it is not scoped here. */
export async function syncCurrentFile (deps: SyncCommandDeps): Promise<void> {
  const editor = vscode.window.activeTextEditor
  const folder = folderForActiveEditor()
  if (!editor || !folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a file inside the workspace to sync it.')

    return
  }

  await syncFile(deps, folder, relative(folder, editor.document.uri.fsPath).replaceAll(/[/\\]/g, '/'))
}

/** Syncs one file, given as a path relative to the workspace folder, the same way. */
export async function syncFile (deps: SyncCommandDeps, folder: string, relativePath: string): Promise<void> {
  await syncOutputs(
    {
      engine:        deps.engine,
      connections:   deps.connections,
      targets:       deps.targets,
      workspaceRoot: folder,
      scope:         [relativePath],
      ui:            createSyncUI(deps.output),
      addConnection: () => connectConfluence({ engine: deps.engine, store: deps.connections }),
    },
    { confluenceOnly: true },
  )
}
