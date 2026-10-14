#!/usr/bin/env node
// Attaches the engine's six platform zips (and a checksum file) to the extension's GitHub
// Release, so a CI pipeline can download a versioned `lore-master-engine` with one `curl`
// (#201). The engine has no tag of its own: it is stamped with, and released as, the version of
// the extension that carries it.
//
//   node tools/engine-release-assets.cjs        attach to the release tagged on HEAD
//
// Runs after `mnci ci release`; does nothing when this run released no extension.
'use strict'
const { spawnSync } = require('node:child_process')
const { createHash } = require('node:crypto')
const { existsSync, readdirSync, readFileSync, writeFileSync } = require('node:fs')
const { join } = require('node:path')

const ENGINE = 'lore-master-engine'
const EXTENSION = 'lore-master-vscode'
const DROP = 'dist/drop'

function fail (message) {
  console.error(message)
  process.exit(1)
}

function run (command, args, options = {}) {
  const result = spawnSync(command, args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'], shell: process.platform === 'win32', ...options })
  if (result.status !== 0) fail(command + ' ' + args.join(' ') + ' failed (exit ' + result.status + ')')

  return result.stdout
}

const tag = run('git', ['tag', '--points-at', 'HEAD']).split(/\r?\n/).find(each => each.startsWith(EXTENSION + '@'))
if (!tag) {
  console.log('This run released no ' + EXTENSION + ' version - no engine zips to attach.')
  process.exit(0)
}
const version = tag.slice(EXTENSION.length + 1)
console.log('Building the six ' + ENGINE + ' platforms as ' + version)
run('npx', ['nx', 'run', ENGINE + ':package-all'], { env: { ...process.env, VERSION: version }, stdio: 'inherit' })

const prefix = 'go-app-' + ENGINE + '-'
const zips = existsSync(DROP) ? readdirSync(DROP).filter(each => each.startsWith(prefix) && each.endsWith('.zip')).sort((a, b) => a.localeCompare(b)) : []
if (zips.length !== 6) fail('Expected six ' + prefix + '*.zip in ' + DROP + ', found ' + zips.length + '.')

const sums = zips.map(name => createHash('sha256').update(readFileSync(join(DROP, name))).digest('hex') + '  ' + name).join('\n') + '\n'
const sumsPath = join(DROP, ENGINE + '-' + version + '-sha256.txt')
writeFileSync(sumsPath, sums)

run('gh', ['release', 'upload', tag, ...zips.map(name => join(DROP, name)), sumsPath, '--clobber'], { stdio: 'inherit' })
console.log('Attached ' + zips.length + ' zips and the checksums to ' + tag)
