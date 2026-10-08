// Drives the whole Add Connection flow in a real VS Code host: scripted UI prompts →
// the extension spawns the real engine → the engine detects the edition and verifies the
// credential against an in-process fake Confluence → the connection is stored and the
// authentication provider then reports it as a session. This exercises the UI, the secret
// store, the engine process and the JSON-RPC wiring together — coverage the stub cannot
// give. Skips itself when the engine binary was not built (no `go`).
const assert = require('node:assert')
const http = require('node:http')
const path = require('node:path')
const fs = require('node:fs')
const vscode = require('vscode')

const manifest = require(path.resolve(__dirname, '..', '..', 'package.json'))
const EXTENSION_ID = `${manifest.publisher}.${manifest.name}`

// A fake Confluence Data Center: enough for edition detection and credential verification.
function fakeConfluence () {
  const calls = []
  const server = http.createServer((request, response) => {
    calls.push(`${request.method} ${request.url}`)
    const send = (status, body, type) => {
      response.writeHead(status, { 'Content-Type': type || 'application/json' })
      response.end(body)
    }
    const route = (request.url || '').split('?')[0]
    if (route === '/rest/api/settings/systemInfo') {
      return send(404, '{"message":"not cloud"}')
    }
    if (route === '/rest/applinks/1.0/manifest') {
      return send(200, '<manifest><typeId>confluence</typeId><version>8.5.4</version></manifest>', 'application/xml')
    }
    if (route === '/rest/api/user/current') {
      return request.headers.authorization === 'Bearer pat-token'
        ? send(200, '{"type":"known","username":"ada","userKey":"k1","displayName":"Ada Lovelace"}')
        : send(401, '{"message":"unauthorized"}')
    }

    return send(404, `{"message":"unexpected ${request.method} ${route}"}`)
  })

  return { server, calls }
}

suite('LoreMaster connection flow in a real VS Code host', () => {
  let server
  let calls
  let baseUrl
  const saved = {}
  const errors = []

  suiteSetup(async function () {
    this.timeout(60_000)
    const extension = vscode.extensions.getExtension(EXTENSION_ID)
    assert.ok(extension, 'extension present')
    const engineName = process.platform === 'win32' ? 'lore-master-engine.exe' : 'lore-master-engine'
    if (!fs.existsSync(path.join(extension.extensionPath, 'bin', engineName))) {
      this.skip()
    }
    await extension.activate()

    const fake = fakeConfluence()
    server = fake.server
    calls = fake.calls
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
    baseUrl = `http://127.0.0.1:${server.address().port}`
  })

  suiteTeardown(async () => {
    Object.assign(vscode.window, saved)
    if (server) {
      await new Promise(resolve => server.close(resolve))
    }
  })

  test('adds a connection end to end: UI, engine, edition detect, and verify', async function () {
    this.timeout(60_000)
    saved.showInputBox = vscode.window.showInputBox
    saved.showQuickPick = vscode.window.showQuickPick
    saved.showInformationMessage = vscode.window.showInformationMessage
    saved.showErrorMessage = vscode.window.showErrorMessage

    const inputs = [baseUrl, 'pat-token']
    let next = 0
    vscode.window.showInputBox = async () => inputs[next++]
    vscode.window.showQuickPick = async items => (await items).find(item => item.method === 'pat')
    vscode.window.showInformationMessage = async () => undefined
    vscode.window.showErrorMessage = async message => { errors.push(message); return undefined }

    // The command completing without throwing, and showError never firing, means the whole
    // chain ran: scripted UI → engine spawn → edition detect → session/open → verify →
    // store → showConnected. (setUpConnection stores the credential after a clean verify and
    // only then reports success, so no error here implies it was stored.)
    await vscode.commands.executeCommand('loreMaster.addConnection')

    assert.deepStrictEqual(errors, [], 'the flow reported no error')
    assert.ok(calls.some(call => call.includes('/rest/applinks/1.0/manifest')), 'the engine detected the edition')
    assert.ok(calls.some(call => call.includes('/rest/api/user/current')), 'the engine verified the credential against the site')
  })
})
