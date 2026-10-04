// The Mocha entry VS Code's test host calls: it collects the test files and runs them,
// resolving when they pass and rejecting on any failure so run.js exits non-zero.
const path = require('node:path')
const Mocha = require('mocha')

function run () {
  const mocha = new Mocha({ ui: 'tdd', color: false, timeout: 60_000 })
  mocha.addFile(path.resolve(__dirname, 'host-checks.js'))
  mocha.addFile(path.resolve(__dirname, 'connection-flow.js'))

  return new Promise((resolve, reject) => {
    try {
      mocha.run(failures => {
        if (failures > 0) {
          reject(new Error(`${failures} integration test(s) failed`))
        } else {
          resolve()
        }
      })
    } catch (error) {
      reject(error)
    }
  })
}

module.exports = { run }
