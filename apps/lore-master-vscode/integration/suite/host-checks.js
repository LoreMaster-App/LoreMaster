// Runs inside the VS Code extension host against the REAL vscode API — the coverage the
// unit tests (which use a stub) can't give: that the extension activates, registers its
// commands, and registers the Confluence authentication provider in a true runtime.
const assert = require('node:assert')
const vscode = require('vscode')

const EXTENSION_ID = 'russoedu.lore-master'
const AUTH_PROVIDER_ID = 'lore-master-confluence'
const COMMANDS = ['loreMaster.syncWorkspace', 'loreMaster.syncCurrentFile', 'loreMaster.addConnection']

suite('Lore Master in a real VS Code host', () => {
  suiteSetup(async function () {
    this.timeout(60_000)
    const extension = vscode.extensions.getExtension(EXTENSION_ID)
    assert.ok(extension, `extension ${EXTENSION_ID} is loaded`)
    await extension.activate()
    assert.ok(extension.isActive, 'extension activated without error')
  })

  test('contributes its commands', async () => {
    const commands = await vscode.commands.getCommands(true)
    for (const id of COMMANDS) {
      assert.ok(commands.includes(id), `command ${id} is registered`)
    }
  })

  test('registers the Confluence authentication provider', async () => {
    // For a registered provider with no stored session this resolves to undefined; for an
    // unregistered provider id VS Code rejects — so resolving proves the provider is there.
    const session = await vscode.authentication.getSession(AUTH_PROVIDER_ID, [], { createIfNone: false, silent: true })
    assert.strictEqual(session, undefined, 'the provider answered (no session yet)')
  })
})
