import * as vscode from 'vscode'
import { ADD_CONNECTION_COMMAND, createConnectionUI, registerConfluenceAuth, setUpConnection } from './connection-setup'
import { answerRenderDiagrams, createMermaidRenderer } from './diagram-rendering'
import { createEngineClient, resolveEngineBinary } from './engine-process'
import { COPY_MCP_CONFIG_COMMAND, copyMcpConfig, registerMcpServer } from './mcp-server'
import { PUBLISH_PAGES_COMMAND, publishPagesCommand } from './pages-command'
import { registerPagesView } from './pages-view'
import { createConnectionStore } from './secret-storage'
import { ADD_STORAGE_COMMAND, addStorageCommand, OPEN_CONFIG_COMMAND, openConfig, REFRESH_STORAGES_COMMAND, REMOVE_STORAGE_COMMAND, registerSyncView, removeStorageCommand, STORAGES_VIEW_ID, type StorageNode, StoragesViewProvider } from './sidebar'
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

  // Answer the engine's host/renderDiagram with a Mermaid webview (image mode); and surface
  // Confluence connections in VS Code's Accounts menu (sign in, see the account, sign out),
  // bridged to the same connection store the commands use.
  const renderer = createMermaidRenderer(context.extensionUri)
  context.subscriptions.push(
    renderer,
    answerRenderDiagrams({ engine, renderer }),
    registerConfluenceAuth({ store: connections, signIn: () => setUpConnection({ engine, store: connections, ui: createConnectionUI() }) }),
    // Expose the engine's MCP server so the editor's AI agent knows LoreMaster's rules.
    registerMcpServer(context),
  )

  const syncDeps = { engine, connections, targets, output }
  const storages = new StoragesViewProvider(engine)

  context.subscriptions.push(
    registerSyncView(),
    registerPagesView(syncDeps),
    vscode.window.registerTreeDataProvider(STORAGES_VIEW_ID, storages),
    vscode.commands.registerCommand(SYNC_COMMAND, () => sync(syncDeps)),
    vscode.commands.registerCommand(SYNC_TO_COMMAND, () => syncTo(syncDeps)),
    vscode.commands.registerCommand(SYNC_CURRENT_FILE_COMMAND, () => syncCurrentFile(syncDeps)),
    vscode.commands.registerCommand(PUBLISH_PAGES_COMMAND, () => publishPagesCommand({ engine, output })),
    vscode.commands.registerCommand(ADD_STORAGE_COMMAND, async () => { await addStorageCommand({ engine, connections, output }); storages.refresh() }),
    vscode.commands.registerCommand(OPEN_CONFIG_COMMAND, () => openConfig()),
    vscode.commands.registerCommand(REMOVE_STORAGE_COMMAND, async (node: StorageNode | undefined) => { await removeStorageCommand({ engine }, node); storages.refresh() }),
    vscode.commands.registerCommand(REFRESH_STORAGES_COMMAND, () => storages.refresh()),
    vscode.commands.registerCommand(ADD_CONNECTION_COMMAND, () => setUpConnection({ engine, store: connections, ui: createConnectionUI() })),
    vscode.commands.registerCommand(COPY_MCP_CONFIG_COMMAND, () => copyMcpConfig(context)),
  )
}

export function deactivate (): void {}
