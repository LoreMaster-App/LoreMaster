// Launches a real VS Code with the built extension loaded and runs the Mocha suite in the
// extension host (so the tests see the real `vscode` API, not the unit-test stub). This is
// how the Accounts-menu authentication provider and command registration are verified in a
// true runtime. Run it with `node apps/lore-master-vscode/integration/run.js` after building
// the extension (`npx nx build lore-master-vscode`). It downloads a VS Code build on first
// run and caches it under .vscode-test/.
const path = require('node:path')
const { runTests } = require('@vscode/test-electron')

async function main () {
  // The extension folder: its package.json `main` points at dist/main.js, which the build
  // produces, so the host loads the real bundle.
  const extensionDevelopmentPath = path.resolve(__dirname, '..')
  const extensionTestsPath = path.resolve(__dirname, 'suite', 'index.js')

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
