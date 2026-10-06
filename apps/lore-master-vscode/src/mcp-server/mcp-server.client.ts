import * as vscode from 'vscode'
import { resolveEngineBinary } from '../engine-process'

/** The id shared with package.json's contributes.mcpServerDefinitionProviders entry. */
export const MCP_PROVIDER_ID = 'loreMaster.engine'

/**
 * Registers the bundled engine's `--mcp` mode as an MCP server, so the editor's AI agent
 * learns LoreMaster's documentation rules with no setup. The server points at the same
 * engine binary the sync uses and at the open workspace; it is re-advertised when the
 * workspace folders change. The returned disposable tears everything down on deactivate.
 */
export function registerMcpServer (context: vscode.ExtensionContext): vscode.Disposable {
  const changed = new vscode.EventEmitter<void>()
  const folders = vscode.workspace.onDidChangeWorkspaceFolders(() => changed.fire())

  const provider: vscode.McpServerDefinitionProvider = {
    onDidChangeMcpServerDefinitions: changed.event,
    provideMcpServerDefinitions:     () => [buildServerDefinition(context)],
  }
  const registration = vscode.lm.registerMcpServerDefinitionProvider(MCP_PROVIDER_ID, provider)

  return vscode.Disposable.from(registration, folders, changed)
}

/**
 * Builds the stdio server definition: the resolved engine binary launched with `--mcp` and
 * the open workspace, so the workspace-aware tools (preview_tree, validate_document,
 * place_document) read the right project.
 */
export function buildServerDefinition (context: vscode.ExtensionContext): vscode.McpStdioServerDefinition {
  const command = resolveEngineBinary({
    platform:       process.platform,
    extensionPath:  context.extensionPath,
    configuredPath: vscode.workspace.getConfiguration('loreMaster').get<string>('engine.path'),
  })
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath

  const args = ['--mcp']
  if (workspaceRoot !== undefined) {
    args.push('--workspace', workspaceRoot)
  }

  const definition = new vscode.McpStdioServerDefinition('LoreMaster', command, args)
  if (workspaceRoot !== undefined) {
    definition.cwd = vscode.Uri.file(workspaceRoot)
  }

  return definition
}
