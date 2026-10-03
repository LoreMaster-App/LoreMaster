import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { ConnectionStore } from '../secret-storage'
import type { TargetStore } from '../sync-target'
import { pickWorkspaceFolder } from '../workspace-files'
import { runSync } from './sync-run.use-case'
import { createSyncUI } from './sync-ui.client'

/** The command id contributed in package.json. */
export const SYNC_WORKSPACE_COMMAND = 'loreMaster.syncWorkspace'

/** What the sync commands are given at registration. */
export interface SyncCommandDeps {
  engine:      EngineClient
  connections: ConnectionStore
  targets:     TargetStore
  output:      vscode.OutputChannel
}

/** Syncs the whole open workspace folder. */
export async function syncWorkspace (deps: SyncCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('Lore Master: open a folder to sync.')

    return
  }

  await runSync({ engine: deps.engine, connections: deps.connections, targets: deps.targets, workspaceRoot: folder, ui: createSyncUI(deps.output) })
}
