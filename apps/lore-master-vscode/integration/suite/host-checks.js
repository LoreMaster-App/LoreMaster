// Runs inside the VS Code extension host against the REAL vscode API — the coverage the
// unit tests (which use a stub) can't give: that the extension activates, registers everything
// its manifest contributes, and registers the Confluence authentication provider in a true
// runtime. The extension id, commands and views come from package.json, so a rename or a new
// contribution cannot leave this suite behind.
const assert = require('node:assert')
const path = require('node:path')
const vscode = require('vscode')

const manifest = require(path.resolve(__dirname, '..', '..', 'package.json'))
const EXTENSION_ID = `${manifest.publisher}.${manifest.name}`
const AUTH_PROVIDER_ID = 'lore-master-confluence'
const COMMANDS = manifest.contributes.commands.map(command => command.command)
const VIEWS = Object.values(manifest.contributes.views).flat().map(view => view.id)

suite('LoreMaster in a real VS Code host', () => {
  suiteSetup(async function () {
    this.timeout(60_000)
    const extension = vscode.extensions.getExtension(EXTENSION_ID)
    assert.ok(extension, `extension ${EXTENSION_ID} is loaded`)
    await extension.activate()
    assert.ok(extension.isActive, 'extension activated without error')
  })

  test('registers every command its manifest contributes', async () => {
    const registered = await vscode.commands.getCommands(true)
    for (const id of COMMANDS) {
      assert.ok(registered.includes(id), `command ${id} is registered`)
    }
  })

  test('contributes its sidebar views, including the Pages view', async () => {
    assert.ok(VIEWS.includes('loreMaster.pages'), 'the manifest contributes the Pages view')
    // VS Code adds a "<view id>.focus" command for every view it knows.
    const registered = await vscode.commands.getCommands(true)
    for (const id of VIEWS) {
      assert.ok(registered.includes(`${id}.focus`), `view ${id} is contributed`)
    }
  })

  test('declares the Confluence authentication provider in its manifest', () => {
    // VS Code warns, and a future version will refuse, when a provider is registered but not declared.
    const declared = (manifest.contributes.authentication ?? []).map(provider => provider.id)
    assert.ok(declared.includes(AUTH_PROVIDER_ID), `${AUTH_PROVIDER_ID} is declared under contributes.authentication`)
  })

  test('registers the Confluence authentication provider', async () => {
    // For a registered provider with no stored session this resolves to undefined; for an
    // unregistered provider id VS Code rejects — so resolving proves the provider is there.
    const session = await vscode.authentication.getSession(AUTH_PROVIDER_ID, [], { createIfNone: false, silent: true })
    assert.strictEqual(session, undefined, 'the provider answered (no session yet)')
  })

  test('contributes the page label setting and the toggle flips it', async () => {
    const configuration = () => vscode.workspace.getConfiguration('loreMaster')
    assert.strictEqual(configuration().inspect('pages.label')?.defaultValue, 'title', 'the setting is contributed, defaulting to the content title')

    await vscode.commands.executeCommand('loreMaster.togglePageLabel')
    assert.strictEqual(configuration().get('pages.label'), 'fileName')
    await vscode.commands.executeCommand('loreMaster.togglePageLabel')
    assert.strictEqual(configuration().get('pages.label'), 'title')
    await configuration().update('pages.label', undefined, vscode.ConfigurationTarget.Global)
  })

  test('refreshing the Pages view with no workspace does not fail', async () => {
    await vscode.commands.executeCommand('loreMaster.refreshPages')
  })
})
