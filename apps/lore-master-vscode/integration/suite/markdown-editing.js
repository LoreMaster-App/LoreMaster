// The Markdown editing commands run against a real document in a real VS Code: the unit tests cover the text
// algorithms, and this covers what only a host can: that the commands find the active editor, apply their
// edits and move the cursors.
const assert = require('node:assert')
const path = require('node:path')
const vscode = require('vscode')

const manifest = require(path.resolve(__dirname, '..', '..', 'package.json'))
const EXTENSION_ID = `${manifest.publisher}.${manifest.name}`

async function open (content) {
  const document = await vscode.workspace.openTextDocument({ language: 'markdown', content })
  const editor = await vscode.window.showTextDocument(document)

  return { document, editor }
}

const select = (editor, ...ranges) => {
  editor.selections = ranges.map(([line, from, toLine, to]) => new vscode.Selection(line, from, toLine ?? line, to ?? from))
}

suite('Markdown editing in a real VS Code host', () => {
  suiteSetup(async function () {
    this.timeout(60_000)
    await vscode.extensions.getExtension(EXTENSION_ID).activate()
  })

  teardown(() => vscode.commands.executeCommand('workbench.action.closeAllEditors'))

  test('toggles bold on a selection and off again, keeping it selected', async () => {
    const { document, editor } = await open('hello world')
    select(editor, [0, 0, 0, 5])

    await vscode.commands.executeCommand('loreMaster.markdown.toggleBold')
    assert.strictEqual(document.getText(), '**hello** world')
    assert.strictEqual(document.getText(editor.selection), 'hello')

    await vscode.commands.executeCommand('loreMaster.markdown.toggleBold')
    assert.strictEqual(document.getText(), 'hello world')
  })

  test('formats every cursor, and moves each past the edits above it', async () => {
    const { document, editor } = await open('one two')
    select(editor, [0, 1], [0, 5])

    await vscode.commands.executeCommand('loreMaster.markdown.toggleItalic')

    assert.strictEqual(document.getText(), '*one* *two*')
    assert.deepStrictEqual(editor.selections.map(selection => [selection.active.line, selection.active.character]), [[0, 2], [0, 8]])
  })

  test('cycles a line through a task list', async () => {
    const { document } = await open('buy milk')

    await vscode.commands.executeCommand('loreMaster.markdown.toggleTask')
    assert.strictEqual(document.getText(), '- [ ] buy milk')
    await vscode.commands.executeCommand('loreMaster.markdown.toggleTask')
    assert.strictEqual(document.getText(), '- [x] buy milk')
  })

  test('Enter in a list starts the next numbered item, and on an empty one ends the list', async () => {
    const { document, editor } = await open('1. one\n2. two')
    select(editor, [1, 6])

    await vscode.commands.executeCommand('loreMaster.markdown.listEnter')
    assert.strictEqual(document.getText(), '1. one\n2. two\n3. ')
    assert.deepStrictEqual([editor.selection.active.line, editor.selection.active.character], [2, 3])

    await vscode.commands.executeCommand('loreMaster.markdown.listEnter')
    assert.strictEqual(document.getText(), '1. one\n2. two\n')
  })

  test('Tab nests a list item and numbers the list again', async () => {
    const { document, editor } = await open('1. a\n2. b\n3. c')
    select(editor, [1, 4])

    await vscode.commands.executeCommand('loreMaster.markdown.listIndent')

    assert.strictEqual(document.getText(), '1. a\n   1. b\n2. c')
  })

  test('Tab in a table lines it up and moves to the next cell, and adds a row after the last', async () => {
    const { document, editor } = await open('| a | b |\n|---|---|\n| longer | x |')
    select(editor, [0, 3])

    await vscode.commands.executeCommand('loreMaster.markdown.tableNextCell')
    assert.strictEqual(document.lineAt(0).text, '| a      | b   |')
    assert.deepStrictEqual([editor.selection.active.line, editor.selection.active.character], [0, 11])

    select(editor, [2, 12])
    await vscode.commands.executeCommand('loreMaster.markdown.tableNextCell')
    assert.strictEqual(document.lineCount, 4)
    assert.deepStrictEqual([editor.selection.active.line, editor.selection.active.character], [3, 2])
  })

  test('writes a table of contents and keeps it in step with the headings', async () => {
    const { document, editor } = await open('# Guide\n\n## One\n\n## Two')
    select(editor, [1, 0])

    await vscode.commands.executeCommand('loreMaster.markdown.insertToc')
    assert.ok(document.getText().includes('<!-- toc -->\n- [One](#one)\n- [Two](#two)\n<!-- /toc -->'), document.getText())

    await editor.edit(builder => builder.insert(new vscode.Position(document.lineCount, 0), '\n## Three\n'))
    await vscode.commands.executeCommand('loreMaster.markdown.updateToc')
    assert.ok(document.getText().includes('- [Three](#three)\n<!-- /toc -->'), document.getText())
  })

  test('numbers the headings and takes the numbers off again', async () => {
    const { document } = await open('# Guide\n## A\n### A1\n## B')

    await vscode.commands.executeCommand('loreMaster.markdown.addSectionNumbers')
    assert.strictEqual(document.getText(), '# Guide\n## 1 A\n### 1.1 A1\n## 2 B')

    await vscode.commands.executeCommand('loreMaster.markdown.removeSectionNumbers')
    assert.strictEqual(document.getText(), '# Guide\n## A\n### A1\n## B')
  })

  test('opens the preview beside the editor and follows the file', async () => {
    const { document } = await open('# Preview me\n\nSome **text**.')
    const previewTabs = () => vscode.window.tabGroups.all.flatMap(group => group.tabs).filter(tab => tab.input instanceof vscode.TabInputWebview && tab.input.viewType.includes('loreMasterMarkdownPreview'))
    // The tab list is updated a moment after the command returns.
    const settled = async (count) => {
      for (let attempt = 0; attempt < 40 && previewTabs().length !== count; attempt += 1) {
        await new Promise(resolve => setTimeout(resolve, 100))
      }
    }

    await vscode.commands.executeCommand('loreMaster.markdown.openPreview')
    await settled(1)
    const tabs = vscode.window.tabGroups.all.flatMap(group => group.tabs).map(tab => `${tab.label} (${tab.input?.constructor?.name}: ${tab.input?.viewType ?? ''})`)
    assert.strictEqual(previewTabs().length, 1, `one preview tab is open; tabs are ${JSON.stringify(tabs)}`)
    assert.ok(previewTabs()[0].label.startsWith('Preview '), previewTabs()[0].label)

    // A second request brings the same preview forward and does not open another one.
    await vscode.window.showTextDocument(document)
    await vscode.commands.executeCommand('loreMaster.markdown.openPreview')
    await new Promise(resolve => setTimeout(resolve, 500))
    assert.strictEqual(previewTabs().length, 1)
  })

  test('does nothing in a file that is not Markdown', async () => {
    const document = await vscode.workspace.openTextDocument({ language: 'plaintext', content: 'hello' })
    const editor = await vscode.window.showTextDocument(document)
    select(editor, [0, 0, 0, 5])

    await vscode.commands.executeCommand('loreMaster.markdown.toggleBold')

    assert.strictEqual(document.getText(), 'hello')
  })
})
