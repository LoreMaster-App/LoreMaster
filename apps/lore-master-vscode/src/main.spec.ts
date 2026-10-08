import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import * as vscode from 'vscode'
import { activate } from './main'

interface Manifest {
  contributes: { commands: { command: string }[] }
}

function fakeContext (): vscode.ExtensionContext {
  return {
    subscriptions: [],
    extensionPath: '/ext',
    extensionUri:  { fsPath: '/ext' },
    secrets:       { get: async () => undefined, store: async () => {}, delete: async () => {} },
    globalState:   { get: (_key: string, value: unknown) => value, update: async () => {} },
  } as unknown as vscode.ExtensionContext
}

describe('activate', () => {
  it('registers every command package.json contributes', async () => {
    const context = fakeContext()
    const manifest = JSON.parse(readFileSync(join(__dirname, '../package.json'), 'utf8')) as Manifest
    const contributed = manifest.contributes.commands.map(command => command.command)

    activate(context)

    expect(contributed).toEqual(['loreMaster.sync', 'loreMaster.syncTo', 'loreMaster.syncCurrentFile', 'loreMaster.publishPages', 'loreMaster.addConnection', 'loreMaster.addStorage', 'loreMaster.openConfig', 'loreMaster.removeStorage', 'loreMaster.refreshStorages', 'loreMaster.copyMcpConfig', 'loreMaster.refreshPages', 'loreMaster.checkRemote', 'loreMaster.togglePageLabel', 'loreMaster.syncPage', 'loreMaster.editStorage', 'loreMaster.runGenerators', 'loreMaster.createAgent', 'loreMaster.addGenerator', 'loreMaster.runGenerator', 'loreMaster.editGenerator', 'loreMaster.removeGenerator', 'loreMaster.refreshGenerators', 'loreMaster.generateAndSync', 'loreMaster.toggleWatch'])
    expect(await vscode.commands.getCommands()).toEqual(expect.arrayContaining(contributed))
    // The commands plus the engine client, output channel, diagram renderer, its
    // host/renderDiagram subscription, the Confluence authentication provider, the MCP
    // server provider and the two sidebar views (Sync, Storages) — all disposed on
    // deactivate. The Pages view and its four commands are one disposable, and so are the
    // Generators view and its six commands (the five it contributes and Run generators).
    const pagesViewCommands = 4
    const generatorsViewCommands = 6
    expect(context.subscriptions).toHaveLength(contributed.length - pagesViewCommands - generatorsViewCommands + 11)
  })
})
