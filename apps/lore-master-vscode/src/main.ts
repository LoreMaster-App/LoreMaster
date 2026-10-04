import * as vscode from 'vscode'
import { ADD_CONNECTION_COMMAND, answerOpenExternal, createConnectionUI, registerConfluenceAuth, setUpConnection } from './connection-setup'
import { answerRenderDiagrams, createMermaidRenderer } from './diagram-rendering'
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

  // Answer the engine's host/renderDiagram with a Mermaid webview (image mode) and its
  // host/openExternal by opening the OAuth page in the browser; and surface Confluence
  // connections in VS Code's Accounts menu (sign in, see the account, sign out), bridged to
  // the same connection store the commands use.
  const renderer = createMermaidRenderer(context.extensionUri)
  context.subscriptions.push(
    renderer,
    answerRenderDiagrams({ engine, renderer }),
    answerOpenExternal({ engine, open: async url => { await vscode.env.openExternal(vscode.Uri.parse(url)) } }),
    registerConfluenceAuth({ store: connections, signIn: () => setUpConnection({ engine, store: connections, ui: createConnectionUI() }) }),
  )

  const syncDeps = { engine, connections, targets, output }

  context.subscriptions.push(
    vscode.commands.registerCommand(SYNC_WORKSPACE_COMMAND, () => syncWorkspace(syncDeps)),
    vscode.commands.registerCommand(SYNC_CURRENT_FILE_COMMAND, () => syncCurrentFile(syncDeps)),
    vscode.commands.registerCommand(ADD_CONNECTION_COMMAND, () => setUpConnection({ engine, store: connections, ui: createConnectionUI() })),
  )
}

export function deactivate (): void {}
