// Launches a real VS Code with the built extension loaded and runs the Mocha suite in the
// extension host (so the tests see the real `vscode` API, not the unit-test stub). This is
// how the Accounts-menu authentication provider and command registration are verified in a
// true runtime. Run it with `node apps/lore-master-vscode/integration/run.js` after building
// the extension (`npx nx build lore-master-vscode`). It downloads a VS Code build on first
// run and caches it under .vscode-test/.
const path = require('node:path')
const fs = require('node:fs')
const { execFileSync } = require('node:child_process')
const { runTests } = require('@vscode/test-electron')

// Builds the engine into the dev extension's bin/ so resolveEngineBinary finds it and the
// connection flow can spawn it. Best-effort: without `go` the connection test skips itself,
// but activation/commands/provider still run.
function buildEngine (extensionDevelopmentPath) {
  try {
    const name = process.platform === 'win32' ? 'lore-master-engine.exe' : 'lore-master-engine'
    const binDir = path.join(extensionDevelopmentPath, 'bin')
    fs.mkdirSync(binDir, { recursive: true })
    const repoRoot = path.resolve(extensionDevelopmentPath, '..', '..')
    execFileSync('go', ['build', '-o', path.join(binDir, name), './apps/lore-master-engine'], { cwd: repoRoot, stdio: 'inherit' })
    console.log('built engine into', binDir)
  } catch (error) {
    console.warn('could not build the engine (the connection flow test will skip):', error.message)
  }
}

async function main () {
  // The extension folder: its package.json `main` points at dist/main.js, which the build
  // produces, so the host loads the real bundle.
  const extensionDevelopmentPath = path.resolve(__dirname, '..')
  const extensionTestsPath = path.resolve(__dirname, 'suite', 'index.js')

  buildEngine(extensionDevelopmentPath)

  await runTests({
    extensionDevelopmentPath,
    extensionTestsPath,
    // Isolate from the user's installed extensions and profile.
    launchArgs: ['--disable-extensions', '--disable-gpu'],
  })
}

main().catch(error => {
  console.error('integration test run failed:', error)
  process.exit(1)
})
