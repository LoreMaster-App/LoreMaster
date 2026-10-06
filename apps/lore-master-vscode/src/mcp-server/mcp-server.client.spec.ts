import * as vscode from 'vscode'
import { buildServerDefinition, MCP_PROVIDER_ID, registerMcpServer } from './mcp-server.client'

// The 'vscode' module is the test stub at runtime; these are the stub-only surfaces this
// test drives (a mutable workspaceFolders and the registered MCP provider), which the real
// @types/vscode does not expose.
interface StubVscode {
  workspace: { workspaceFolders: { uri: { fsPath: string }; name: string }[] | undefined }
  mcpProvider (id: string): { provideMcpServerDefinitions (): { command: string }[] } | undefined
}
const stub = vscode as unknown as StubVscode

function fakeContext (): vscode.ExtensionContext {
  return { extensionPath: '/ext', extensionUri: { fsPath: '/ext' } } as unknown as vscode.ExtensionContext
}

describe('registerMcpServer', () => {
  afterEach(() => { stub.workspace.workspaceFolders = undefined })

  it('launches the bundled engine with --mcp and the open workspace', () => {
    stub.workspace.workspaceFolders = [{ uri: { fsPath: '/work/space' }, name: 'space' }]

    const definition = buildServerDefinition(fakeContext())

    expect(definition.command).toBeTruthy()
    expect(definition.args[0]).toBe('--mcp')
    expect(definition.args).toContain('--workspace')
    expect(definition.args).toContain('/work/space')
    expect(definition.cwd?.fsPath).toBe('/work/space')
  })

  it('omits --workspace when no folder is open', () => {
    stub.workspace.workspaceFolders = undefined

    const definition = buildServerDefinition(fakeContext())

    expect(definition.args).toEqual(['--mcp'])
    expect(definition.cwd).toBeUndefined()
  })

  it('registers a provider that advertises the server, and disposes cleanly', () => {
    const disposable = registerMcpServer(fakeContext())

    const provider = stub.mcpProvider(MCP_PROVIDER_ID)
    expect(provider).toBeDefined()
    const definitions = provider?.provideMcpServerDefinitions() ?? []
    expect(definitions).toHaveLength(1)
    expect(definitions[0].command).toBeTruthy()

    disposable.dispose()
    expect(stub.mcpProvider(MCP_PROVIDER_ID)).toBeUndefined()
  })
})
