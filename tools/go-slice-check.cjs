#!/usr/bin/env node
// Written by MoNecromanCI; 'mnci upgrade' rewrites it, so local edits do not survive.
//
// node tools/go-slice-check.cjs <project directory>...
//
// The Go counterpart of the vertical-slices/file-role lint rule: a file is <snake>_<role>.go, a test is
// <snake>_<role>_test.go, only doc.go and main.go sit at the root of a module, and no folder is a junk drawer
// (util, helper, common, shared, manager, processor, data, misc). Exits 1 and names every violation.
'use strict'
const { readdirSync } = require('node:fs')
const { join, relative } = require('node:path')

const ROLES = ['handler', 'use_case', 'algorithm', 'policy', 'model', 'contract', 'mapper', 'validator', 'repository', 'client', 'store', 'middleware', 'error', 'config', 'enum']
const PLATFORMS = ['windows', 'linux', 'darwin', 'freebsd', 'js', 'wasm', 'amd64', 'arm64', '386', 'arm']
const FORBIDDEN_FOLDERS = new Set(['util', 'utils', 'helper', 'helpers', 'common', 'shared', 'manager', 'processor', 'data', 'misc'])
const SKIPPED_FOLDERS = new Set(['vendor', 'testdata', 'node_modules', 'dist'])

const platform = '(?:_(?:' + PLATFORMS.join('|') + '))*'
const ROLE_FILE = new RegExp('^[a-z][a-z0-9]*(?:_[a-z0-9]+)*_(?:' + ROLES.join('|') + ')' + platform + '(?:_test)?$')
const ROOT_FILE = new RegExp('^(?:doc|main)' + platform + '(?:_test)?$')
const DOC_FILE = /^doc\.go$/

function check (root) {
  const problems = []
  function walk (directory) {
    const entries = readdirSync(directory, { withFileTypes: true })
    for (const entry of entries) {
      const path = join(directory, entry.name)
      const shown = relative(root, path).replaceAll('\\', '/')
      if (entry.isDirectory()) {
        if (entry.name.startsWith('.') || SKIPPED_FOLDERS.has(entry.name)) continue
        if (FORBIDDEN_FOLDERS.has(entry.name)) {
          problems.push(shown + '/ - "' + entry.name + '" is a junk drawer; name the folder for the outcome it serves')
          continue
        }
        walk(path)
      } else if (entry.name.endsWith('.go')) {
        const stem = entry.name.slice(0, -'.go'.length)
        if (directory === root) {
          if (!ROOT_FILE.test(stem)) problems.push(shown + ' - only doc.go and main.go sit at the root of a module; move this into a slice folder')
        } else if (!ROLE_FILE.test(stem) && !DOC_FILE.test(entry.name)) {
          problems.push(shown + ' - a Go file is <snake>_<role>.go (roles: ' + ROLES.join(', ') + ')')
        }
      }
    }
  }
  walk(root)

  return problems
}

const projects = process.argv.slice(2)
if (projects.length === 0) {
  console.error('Usage: node tools/go-slice-check.cjs <project directory>...')
  process.exit(1)
}
let failed = 0
for (const project of projects) {
  const problems = check(project)
  for (const problem of problems) console.error(project + '/' + problem)
  failed += problems.length
}
if (failed > 0) {
  console.error(failed + ' file(s) break the vertical-slice layout. See the vertical-slice rules; an exception is written in the project README.')
  process.exit(1)
}
console.log('Go slice layout OK: ' + projects.join(', '))
