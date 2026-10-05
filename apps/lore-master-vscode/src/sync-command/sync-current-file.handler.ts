import { relative } from 'node:path'
import * as vscode from 'vscode'
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

  const scope = [relative(folder, editor.document.uri.fsPath).replaceAll(/[/\\]/g, '/')]

  await syncOutputs(
    { engine: deps.engine, connections: deps.connections, targets: deps.targets, workspaceRoot: folder, scope, ui: createSyncUI(deps.output) },
    { confluenceOnly: true },
  )
}
