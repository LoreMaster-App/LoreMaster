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

    expect(contributed).toEqual(['loreMaster.sync', 'loreMaster.syncTo', 'loreMaster.syncCurrentFile', 'loreMaster.publishPages', 'loreMaster.exportMarkdown', 'loreMaster.copyMarkdownAs', 'loreMaster.markdown.toggleBold', 'loreMaster.markdown.toggleItalic', 'loreMaster.markdown.toggleStrikethrough', 'loreMaster.markdown.toggleInlineCode', 'loreMaster.markdown.toggleCodeBlock', 'loreMaster.markdown.toggleQuote', 'loreMaster.markdown.toggleTask', 'loreMaster.markdown.increaseHeading', 'loreMaster.markdown.decreaseHeading', 'loreMaster.markdown.listEnter', 'loreMaster.markdown.listIndent', 'loreMaster.markdown.listOutdent', 'loreMaster.markdown.formatTable', 'loreMaster.markdown.tableNextCell', 'loreMaster.markdown.tablePreviousCell', 'loreMaster.markdown.tableInsertRowAbove', 'loreMaster.markdown.tableInsertRowBelow', 'loreMaster.markdown.tableDeleteRow', 'loreMaster.markdown.tableInsertColumnLeft', 'loreMaster.markdown.tableInsertColumnRight', 'loreMaster.markdown.tableDeleteColumn', 'loreMaster.markdown.insertToc', 'loreMaster.markdown.updateToc', 'loreMaster.markdown.addSectionNumbers', 'loreMaster.markdown.removeSectionNumbers', 'loreMaster.addConnection', 'loreMaster.addStorage', 'loreMaster.openConfig', 'loreMaster.removeStorage', 'loreMaster.refreshStorages', 'loreMaster.copyMcpConfig', 'loreMaster.refreshPages', 'loreMaster.checkRemote', 'loreMaster.togglePageLabel', 'loreMaster.syncPage', 'loreMaster.editStorage', 'loreMaster.runGenerators', 'loreMaster.createAgent', 'loreMaster.setUpCi', 'loreMaster.rollbackSync', 'loreMaster.addGenerator', 'loreMaster.runGenerator', 'loreMaster.editGenerator', 'loreMaster.removeGenerator', 'loreMaster.refreshGenerators', 'loreMaster.generateAndSync', 'loreMaster.toggleWatch', 'loreMaster.choosePagesView', 'loreMaster.togglePagesExcluded', 'loreMaster.excludeFromSync', 'loreMaster.includeInSync'])
    expect(await vscode.commands.getCommands()).toEqual(expect.arrayContaining(contributed))
    // The commands plus the engine client, output channel, diagram renderer, its
    // host/renderDiagram subscription, the Confluence authentication provider, the MCP
    // server provider, the file-move handler and the two sidebar views (Sync, Storages) — all
    // disposed on deactivate. The Pages view and its four commands are one disposable, and so are the
    // Generators view and its six commands (the five it contributes and Run generators).
    const pagesViewCommands = 8
    const generatorsViewCommands = 6
    // Export and Copy As are one disposable too.
    const exportCommands = 2
    // The Markdown editing commands are four disposables: formatting, lists, tables and the table of contents. The diagrams for the built-in preview are a fifth.
    const markdownEditingCommands = 25
    expect(context.subscriptions).toHaveLength(contributed.length - pagesViewCommands - generatorsViewCommands - exportCommands - markdownEditingCommands + 14 + 5)
  })
})
