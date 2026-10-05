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

    expect(contributed).toEqual(['loreMaster.sync', 'loreMaster.syncTo', 'loreMaster.syncCurrentFile', 'loreMaster.publishPages', 'loreMaster.addConnection', 'loreMaster.addStorage', 'loreMaster.openConfig', 'loreMaster.removeStorage', 'loreMaster.refreshStorages'])
    expect(await vscode.commands.getCommands()).toEqual(expect.arrayContaining(contributed))
    // The commands plus the engine client, output channel, diagram renderer, its
    // host/renderDiagram subscription, the host/openExternal subscription, the Confluence
    // authentication provider and the two sidebar views (Sync, Storages) — all disposed on
    // deactivate.
    expect(context.subscriptions).toHaveLength(contributed.length + 8)
  })
})
