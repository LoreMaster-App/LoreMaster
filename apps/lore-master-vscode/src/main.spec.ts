import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import * as vscode from 'vscode'
import { activate } from './main'

interface Manifest {
  contributes: { commands: { command: string }[] }
}

describe('activate', () => {
  it('registers every command package.json contributes', async () => {
    const context = { subscriptions: [] } as unknown as vscode.ExtensionContext
    const manifest = JSON.parse(readFileSync(join(__dirname, '../package.json'), 'utf8')) as Manifest
    const contributed = manifest.contributes.commands.map(command => command.command)

    activate(context)

    expect(contributed).toEqual(['loreMaster.syncWorkspace'])
    expect(context.subscriptions).toHaveLength(contributed.length)
    expect(await vscode.commands.getCommands()).toEqual(expect.arrayContaining(contributed))
  })
})
