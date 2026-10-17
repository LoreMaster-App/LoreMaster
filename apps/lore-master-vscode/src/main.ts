import * as vscode from 'vscode'
import { CREATE_AGENT_COMMAND, createAgentCommand } from './agent-creation'
import { SET_UP_CI_COMMAND, setUpCiCommand } from './ci-setup'
import { ADD_CONNECTION_COMMAND, createConnectionUI, registerConfluenceAuth, setUpConnection } from './connection-setup'
import { answerRenderDiagrams, createMermaidRenderer } from './diagram-rendering'
import { createEngineClient, resolveEngineBinary } from './engine-process'
import { registerExportMarkdown } from './export-markdown'
import { registerFileMoves } from './file-moves'
import { GENERATE_AND_SYNC_COMMAND, generateAndSyncCommand, syncCommand } from './generate-and-sync'
import { registerGeneratorForm } from './generator-form'
import { registerGeneratorsView } from './generators-view'
import { registerMarkdownFormatting } from './markdown-formatting'
import { registerMarkdownLists } from './markdown-lists'
import { registerMarkdownDiagrams, type MarkdownItExtension } from './markdown-diagrams'
import { registerMarkdownTables } from './markdown-tables'
import { registerMarkdownToc } from './markdown-toc'
import { COPY_MCP_CONFIG_COMMAND, copyMcpConfig, registerMcpServer } from './mcp-server'
import { PUBLISH_PAGES_COMMAND, publishPagesCommand } from './pages-command'
import { registerPagesView } from './pages-view'
import { createConnectionStore, createEngineVault, syncWithVault, withKeychain } from './secret-storage'
import { EDIT_STORAGE_COMMAND } from './storage-editing'
import { addStorageForm, editStorageForm } from './storage-form'
import { ROLLBACK_SYNC_COMMAND, rollbackSyncCommand } from './sync-rollback'
import { ADD_STORAGE_COMMAND, OPEN_CONFIG_COMMAND, openConfig, REFRESH_STORAGES_COMMAND, REMOVE_STORAGE_COMMAND, removeStorageCommand, type StorageNode } from './sidebar'
import { SYNC_COMMAND, SYNC_CURRENT_FILE_COMMAND, SYNC_TO_COMMAND, syncCurrentFile, syncTo } from './sync-command'
import { createTargetStore } from './sync-target'
import { TOGGLE_WATCH_COMMAND, WatchMode } from './watch-mode'

/** A multi-root workspace's `.code-workspace` file is a settings layer; the engine reads it
 *  from this variable (a single folder, or a remote file, has none). */
function workspaceFileEnvironment (workspaceFile: vscode.Uri | undefined): Record<string, string> | undefined {
  return workspaceFile?.scheme === 'file' ? { LORE_MASTER_WORKSPACE_FILE: workspaceFile.fsPath } : undefined
}

/** Returns the markdown-it extension VS Code's built-in Markdown preview uses to draw Mermaid and D2. */
export function activate (context: vscode.ExtensionContext): MarkdownItExtension {
  const engine = createEngineClient({
    binaryPath: resolveEngineBinary({
      platform:       process.platform,
      extensionPath:  context.extensionPath,
      configuredPath: vscode.workspace.getConfiguration('loreMaster').get<string>('engine.path'),
    }),
    env: workspaceFileEnvironment(vscode.workspace.workspaceFile),
  })
  context.subscriptions.push(engine)

  const profileConnections = createConnectionStore(context.secrets, context.globalState)
  const targets = createTargetStore()
  const output = vscode.window.createOutputChannel('LoreMaster')
  context.subscriptions.push(output)

  // Opt in: remember connections in the operating system's keychain, shared by every editor
  // profile and by the command line (`lore-master-engine auth`).
  const keychainOn = (): boolean => vscode.workspace.getConfiguration('loreMaster').get<boolean>('credentials.useKeychain', false)
  const vault = createEngineVault(engine)
  const keychainError = (error: unknown): void => { output.appendLine(`The keychain could not be used: ${error instanceof Error ? error.message : String(error)}`) }
  const connections = withKeychain(profileConnections, vault, { enabled: keychainOn, onError: keychainError })
  const shareWithKeychain = (): void => {
    if (keychainOn()) {
      void syncWithVault(profileConnections, vault)
        .then(result => { if (result.imported + result.exported > 0) { output.appendLine(`Keychain: ${result.imported} connection(s) brought in, ${result.exported} remembered.`) } })
        .catch(keychainError)
    }
  }
  shareWithKeychain()
  context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(event => {
    if (event.affectsConfiguration('loreMaster.credentials.useKeychain')) {
      shareWithKeychain()
    }
  }))

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

  const markdownDiagrams = registerMarkdownDiagrams({ engine })
  const syncDeps = { engine, connections, targets, output }
  const watchMode = new WatchMode(syncDeps)

  context.subscriptions.push(
    watchMode,
    registerPagesView(syncDeps, context.workspaceState),
    registerFileMoves({ engine }),
    registerGeneratorsView({ engine, output }, registerGeneratorForm),
    registerExportMarkdown({ engine, output, extensionPath: context.extensionPath }),
    registerMarkdownFormatting(),
    registerMarkdownLists(),
    registerMarkdownTables(),
    registerMarkdownToc(),
    markdownDiagrams,
    vscode.commands.registerCommand(SYNC_COMMAND, () => syncCommand(syncDeps)),
    vscode.commands.registerCommand(GENERATE_AND_SYNC_COMMAND, () => generateAndSyncCommand(syncDeps)),
    vscode.commands.registerCommand(TOGGLE_WATCH_COMMAND, () => watchMode.toggle()),
    vscode.commands.registerCommand(SYNC_TO_COMMAND, () => syncTo(syncDeps)),
    vscode.commands.registerCommand(SYNC_CURRENT_FILE_COMMAND, () => syncCurrentFile(syncDeps)),
    vscode.commands.registerCommand(PUBLISH_PAGES_COMMAND, () => publishPagesCommand({ engine, output })),
    vscode.commands.registerCommand(CREATE_AGENT_COMMAND, () => createAgentCommand({ engine })),
    vscode.commands.registerCommand(ROLLBACK_SYNC_COMMAND, (journalId?: unknown) => rollbackSyncCommand({ engine, connections, output }, typeof journalId === 'string' ? journalId : undefined)),
    vscode.commands.registerCommand(SET_UP_CI_COMMAND, () => setUpCiCommand({ engine, version: String(context.extension.packageJSON.version) })),
    vscode.commands.registerCommand(ADD_STORAGE_COMMAND, () => addStorageForm({ engine, connections })),
    vscode.commands.registerCommand(OPEN_CONFIG_COMMAND, () => openConfig()),
    vscode.commands.registerCommand(REMOVE_STORAGE_COMMAND, async (node: StorageNode | undefined) => { await removeStorageCommand({ engine }, node); await vscode.commands.executeCommand(REFRESH_STORAGES_COMMAND) }),
    vscode.commands.registerCommand(EDIT_STORAGE_COMMAND, (node: StorageNode | undefined) => editStorageForm({ engine, connections }, node)),
    vscode.commands.registerCommand(ADD_CONNECTION_COMMAND, () => setUpConnection({ engine, store: connections, ui: createConnectionUI() })),
    vscode.commands.registerCommand(COPY_MCP_CONFIG_COMMAND, () => copyMcpConfig(context)),
  )

  return markdownDiagrams
}

export function deactivate (): void {}
