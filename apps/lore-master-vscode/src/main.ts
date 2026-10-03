import * as vscode from 'vscode'
import { ADD_CONNECTION_COMMAND, createConnectionUI, setUpConnection } from './connection-setup'
import { createEngineClient, resolveEngineBinary } from './engine-process'
import { createConnectionStore } from './secret-storage'
import { SYNC_CURRENT_FILE_COMMAND, SYNC_WORKSPACE_COMMAND, syncCurrentFile, syncWorkspace } from './sync-command'
import { createTargetStore } from './sync-target'

export function activate (context: vscode.ExtensionContext): void {
  const engine = createEngineClient({
    binaryPath: resolveEngineBinary({
      platform:       process.platform,
      extensionPath:  context.extensionPath,
      configuredPath: vscode.workspace.getConfiguration('loreMaster').get<string>('engine.path'),
    }),
  })
  context.subscriptions.push(engine)

  const connections = createConnectionStore(context.secrets, context.globalState)
  const targets = createTargetStore()
  const output = vscode.window.createOutputChannel('Lore Master')
  context.subscriptions.push(output)

  const syncDeps = { engine, connections, targets, output }

  context.subscriptions.push(
    vscode.commands.registerCommand(SYNC_WORKSPACE_COMMAND, () => syncWorkspace(syncDeps)),
    vscode.commands.registerCommand(SYNC_CURRENT_FILE_COMMAND, () => syncCurrentFile(syncDeps)),
    vscode.commands.registerCommand(ADD_CONNECTION_COMMAND, () => setUpConnection({ engine, store: connections, ui: createConnectionUI() })),
  )
}

export function deactivate (): void {}
