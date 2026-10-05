import * as vscode from 'vscode'
import { connectConfluence } from '../connection-setup'
import type { EngineClient } from '../engine-process'
import type { ConnectionStore } from '../secret-storage'
import { createSyncUI } from '../sync-command'
import { pickWorkspaceFolder } from '../workspace-files'
import { addStorage } from './add-storage.use-case'

/** The command id contributed in package.json. */
export const ADD_STORAGE_COMMAND = 'loreMaster.addStorage'

/** What the sidebar commands are given at registration. */
export interface SidebarCommandDeps {
  engine:      EngineClient
  connections: ConnectionStore
  output:      vscode.OutputChannel
}

/** Sets up one or more new sync storages for the chosen workspace folder. */
export async function addStorageCommand (deps: SidebarCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder first.')

    return
  }

  await addStorage({
    engine:        deps.engine,
    connections:   deps.connections,
    workspaceRoot: folder,
    ui:            createSyncUI(deps.output),
    addConnection: () => connectConfluence({ engine: deps.engine, store: deps.connections }),
  })
}
