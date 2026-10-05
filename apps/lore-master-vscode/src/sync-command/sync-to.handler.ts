import * as vscode from 'vscode'
import { pickWorkspaceFolder } from '../workspace-files'
import type { SyncCommandDeps } from './sync.handler'
import { syncOutputs } from './sync-outputs.use-case'
import { createSyncUI } from './sync-ui.client'

/** The command id contributed in package.json. */
export const SYNC_TO_COMMAND = 'loreMaster.syncTo'

/** Syncs a chosen subset of the workspace's configured outputs. */
export async function syncTo (deps: SyncCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('Lore Master: open a folder to sync.')

    return
  }

  await syncOutputs(
    { engine: deps.engine, connections: deps.connections, targets: deps.targets, workspaceRoot: folder, ui: createSyncUI(deps.output) },
    { choose: true },
  )
}
