import * as vscode from 'vscode'
import { copyMcpConfig } from './copy-mcp-config.use-case'

interface StubWorkspace {
  workspace: { workspaceFolders: { uri: { fsPath: string }; name: string }[] | undefined }
}
const stub = vscode as unknown as StubWorkspace

function fakeContext (): vscode.ExtensionContext {
  return { extensionPath: '/ext', extensionUri: { fsPath: '/ext' } } as unknown as vscode.ExtensionContext
}

describe('copyMcpConfig', () => {
  afterEach(() => { stub.workspace.workspaceFolders = undefined })

  it('copies a valid mcpServers config for the bundled engine and workspace', async () => {
    stub.workspace.workspaceFolders = [{ uri: { fsPath: '/work/space' }, name: 'space' }]

    await copyMcpConfig(fakeContext())

    const copied = JSON.parse(await vscode.env.clipboard.readText()) as {
      mcpServers: { loremaster: { command: string; args: string[] } }
    }
    const server = copied.mcpServers.loremaster
    expect(server.command).toBeTruthy()
    expect(server.args[0]).toBe('--mcp')
    expect(server.args).toContain('--workspace')
    expect(server.args).toContain('/work/space')
  })

  it('still copies a usable config when no folder is open', async () => {
    stub.workspace.workspaceFolders = undefined

    await copyMcpConfig(fakeContext())

    const copied = JSON.parse(await vscode.env.clipboard.readText()) as {
      mcpServers: { loremaster: { args: string[] } }
    }
    expect(copied.mcpServers.loremaster.args).toEqual(['--mcp'])
  })
})
