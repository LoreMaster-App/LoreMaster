import * as vscode from 'vscode'
import { ADD_CONNECTION_COMMAND, createConnectionUI, setUpConnection } from './connection-setup'
import { createEngineClient, resolveEngineBinary } from './engine-process'
import { createConnectionStore } from './secret-storage'
import { SYNC_WORKSPACE_COMMAND, syncWorkspace } from './sync-command'

export function activate (context: vscode.ExtensionContext): void {
  const engine = createEngineClient({
    binaryPath: resolveEngineBinary({
      platform:       process.platform,
      extensionPath:  context.extensionPath,
      configuredPath: vscode.workspace.getConfiguration('loreMaster').get<string>('engine.path'),
    }),
  })
  context.subscriptions.push(engine)

  const store = createConnectionStore(context.secrets, context.globalState)

  context.subscriptions.push(
    vscode.commands.registerCommand(SYNC_WORKSPACE_COMMAND, syncWorkspace),
    vscode.commands.registerCommand(ADD_CONNECTION_COMMAND, () => setUpConnection({ engine, store, ui: createConnectionUI() })),
  )
}

export function deactivate (): void {}
