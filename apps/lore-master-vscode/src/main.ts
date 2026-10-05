import * as vscode from 'vscode'
import { ADD_CONNECTION_COMMAND, answerOpenExternal, createConnectionUI, registerConfluenceAuth, setUpConnection } from './connection-setup'
import { answerRenderDiagrams, createMermaidRenderer } from './diagram-rendering'
import { createEngineClient, resolveEngineBinary } from './engine-process'
import { PUBLISH_PAGES_COMMAND, publishPagesCommand } from './pages-command'
import { createConnectionStore } from './secret-storage'
import { ADD_STORAGE_COMMAND, addStorageCommand, OPEN_CONFIG_COMMAND, openConfig, registerSyncView } from './sidebar'
import { SYNC_COMMAND, SYNC_CURRENT_FILE_COMMAND, SYNC_TO_COMMAND, sync, syncCurrentFile, syncTo } from './sync-command'
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
  const output = vscode.window.createOutputChannel('LoreMaster')
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
    registerSyncView(),
    vscode.commands.registerCommand(SYNC_COMMAND, () => sync(syncDeps)),
    vscode.commands.registerCommand(SYNC_TO_COMMAND, () => syncTo(syncDeps)),
    vscode.commands.registerCommand(SYNC_CURRENT_FILE_COMMAND, () => syncCurrentFile(syncDeps)),
    vscode.commands.registerCommand(PUBLISH_PAGES_COMMAND, () => publishPagesCommand({ engine, output })),
    vscode.commands.registerCommand(ADD_STORAGE_COMMAND, () => addStorageCommand({ engine, connections, output })),
    vscode.commands.registerCommand(OPEN_CONFIG_COMMAND, () => openConfig()),
    vscode.commands.registerCommand(ADD_CONNECTION_COMMAND, () => setUpConnection({ engine, store: connections, ui: createConnectionUI() })),
  )
}

export function deactivate (): void {}
