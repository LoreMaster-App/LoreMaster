import { relative } from 'node:path'
import * as vscode from 'vscode'
import { connectPlatform } from '../connection-setup'
import { folderForActiveEditor, pickWorkspaceFolder } from '../workspace-files'
import type { SyncCommandDeps } from './sync.handler'
import { syncOutputs } from './sync-outputs.use-case'
import { createSyncUI } from './sync-ui.client'

/** The command ids contributed in package.json. */
export const REWRITE_ALL_COMMAND = 'loreMaster.rewriteAll'
export const REWRITE_CURRENT_FILE_COMMAND = 'loreMaster.rewriteCurrentFile'

/** Writes every page of the chosen folder again even when nothing changed, to redo pages that went wrong
 *  (a diagram left as code). A page edited on Confluence is still a conflict. */
export async function rewriteAll (deps: SyncCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to sync.')

    return
  }

  await rewrite(deps, folder)
}

/** Writes the active file's page again even when nothing changed. */
export async function rewriteCurrentFile (deps: SyncCommandDeps): Promise<void> {
  const editor = vscode.window.activeTextEditor
  const folder = folderForActiveEditor()
  if (!editor || !folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a file inside the workspace to rewrite its page.')

    return
  }

  await rewrite(deps, folder, [relative(folder, editor.document.uri.fsPath).replaceAll(/[/\\]/g, '/')])
}

async function rewrite (deps: SyncCommandDeps, folder: string, scope?: string[]): Promise<void> {
  await syncOutputs(
    {
      engine:        deps.engine,
      connections:   deps.connections,
      targets:       deps.targets,
      workspaceRoot: folder,
      scope,
      rewrite:       true,
      ui:            createSyncUI(deps.output),
      addConnection: () => connectPlatform({ engine: deps.engine, store: deps.connections }),
    },
    { pagesOnly: true },
  )
}
