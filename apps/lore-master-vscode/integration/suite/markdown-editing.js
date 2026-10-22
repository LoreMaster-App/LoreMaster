// The Markdown editing commands run against a real document in a real VS Code: the unit tests cover the text
// algorithms, and this covers what only a host can: that the commands find the active editor, apply their
// edits and move the cursors.
const assert = require('node:assert')
const os = require('node:os')
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

  test('the built-in Markdown preview draws mermaid and d2', async function () {
    this.timeout(60_000)
    // The same call VS Code's preview makes to turn Markdown into the page, with this extension's markdown-it plugin in it.
    const render = (markdown) => vscode.commands.executeCommand('markdown.api.render', markdown)

    const mermaid = await render('```mermaid\ngraph TD\n  A-->B\n```')
    // Recent VS Code draws Mermaid itself and the extension steps aside; older ones get the extension's own.
    if (vscode.extensions.getExtension('vscode.mermaid-markdown-features')) {
      assert.ok(mermaid.includes('markdown-mermaid') && !mermaid.includes('loremaster-mermaid'), mermaid)
    } else {
      assert.ok(mermaid.includes('loremaster-mermaid') && mermaid.includes('A--&gt;B'), mermaid)
    }

    // D2 is drawn by the engine, so the first render says so and a later one has the picture.
    const source = '```d2\nclient -> server\n```'
    assert.ok((await render(source)).includes('loremaster-d2'))
    let drawn = ''
    for (let attempt = 0; attempt < 100 && !drawn.includes('<svg'); attempt += 1) {
      await new Promise(resolve => setTimeout(resolve, 200))
      drawn = await render(source)
    }
    assert.ok(drawn.includes('<svg'), drawn)

    const plain = await render('```ts\nlet a = 1\n```')
    assert.ok(!plain.includes('loremaster-diagram'), plain)
  })

  test('the preview of an open file has the action bar, the footnotes and the link marks', async function () {
    this.timeout(60_000)
    const uri = vscode.Uri.file(path.join(os.tmpdir(), 'loremaster-preview-check.md'))
    await vscode.workspace.fs.writeFile(uri, Buffer.from('# Check\n\nA claim[^1] and [a gone file](missing-file.md).\n\n[^1]: The note.\n'))
    try {
      const document = await vscode.workspace.openTextDocument(uri)
      // With a document (not a string) VS Code puts it in the render environment, as the preview does.
      const html = await vscode.commands.executeCommand('markdown.api.render', document)

      assert.ok(html.includes('loremaster-action-bar'), html)
      assert.ok(html.includes('Export...'), html)
      assert.ok(html.includes('loremaster-footnote-ref'), html)
      assert.ok(html.includes('loremaster-link-broken'), html)
      // A note is drawn on its own: the rules of the page (the Mermaid one adds a span) do not run inside it.
      assert.ok(!html.includes('<li id="fn-1"><span'), html)
    } finally {
      await vscode.workspace.fs.delete(uri)
    }
  })

  test('does nothing in a file that is not Markdown', async () => {
    const document = await vscode.workspace.openTextDocument({ language: 'plaintext', content: 'hello' })
    const editor = await vscode.window.showTextDocument(document)
    select(editor, [0, 0, 0, 5])

    await vscode.commands.executeCommand('loreMaster.markdown.toggleBold')

    assert.strictEqual(document.getText(), 'hello')
  })
})
