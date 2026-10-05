import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { ConnectionStore } from '../secret-storage'
import type { TargetStore } from '../sync-target'
import { pickWorkspaceFolder } from '../workspace-files'
import { syncOutputs } from './sync-outputs.use-case'
import { createSyncUI } from './sync-ui.client'

/** The command id contributed in package.json. */
export const SYNC_COMMAND = 'loreMaster.sync'

/** What the sync commands are given at registration. */
export interface SyncCommandDeps {
  engine:      EngineClient
  connections: ConnectionStore
  targets:     TargetStore
  output:      vscode.OutputChannel
}

/** Syncs every configured output of the chosen workspace folder (first run sets them up). */
export async function sync (deps: SyncCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to sync.')

    return
  }

  await syncOutputs({ engine: deps.engine, connections: deps.connections, targets: deps.targets, workspaceRoot: folder, ui: createSyncUI(deps.output) })
}
