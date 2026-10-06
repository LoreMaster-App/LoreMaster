import * as vscode from 'vscode'
import { buildServerDefinition } from './mcp-server.client'

export const COPY_MCP_CONFIG_COMMAND = 'loreMaster.copyMcpConfig'

/**
 * Copies a ready-to-paste MCP server config to the clipboard for an external MCP client
 * (Claude Desktop/Code and others use the `mcpServers` map). It points at the same bundled
 * engine and workspace the in-editor registration uses, so an agent outside the editor gets
 * the same LoreMaster tools. In-editor clients need nothing — the extension registers the
 * server for them.
 */
export async function copyMcpConfig (context: vscode.ExtensionContext): Promise<void> {
  const definition = buildServerDefinition(context)
  const config = {
    mcpServers: {
      loremaster: {
        command: definition.command,
        args:    definition.args,
      },
    },
  }

  await vscode.env.clipboard.writeText(JSON.stringify(config, null, 2))
  await vscode.window.showInformationMessage('LoreMaster: MCP server config copied. Paste it into your MCP client (e.g. Claude Desktop or Claude Code).')
}
