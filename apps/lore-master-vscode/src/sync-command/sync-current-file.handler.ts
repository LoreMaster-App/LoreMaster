import { relative } from 'node:path'
import * as vscode from 'vscode'
import { folderForActiveEditor } from '../workspace-files'
import { runSync } from './sync-run.use-case'
import { createSyncUI } from './sync-ui.client'
import type { SyncCommandDeps } from './sync-workspace.handler'

/** The command id contributed in package.json. */
export const SYNC_CURRENT_FILE_COMMAND = 'loreMaster.syncCurrentFile'

/** Syncs just the active file (and the ancestors it needs), scoping the plan to it. */
export async function syncCurrentFile (deps: SyncCommandDeps): Promise<void> {
  const editor = vscode.window.activeTextEditor
  const folder = folderForActiveEditor()
  if (!editor || !folder) {
    await vscode.window.showInformationMessage('Lore Master: open a file inside the workspace to sync it.')

    return
  }

  const scope = [relative(folder, editor.document.uri.fsPath).replaceAll(/[/\\]/g, '/')]

  await runSync({ engine: deps.engine, connections: deps.connections, targets: deps.targets, workspaceRoot: folder, scope, ui: createSyncUI(deps.output) })
}
