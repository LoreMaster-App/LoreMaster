#!/usr/bin/env node
// A manual smoke check: sync one throwaway page to a real Confluence Data Center space and
// then prune it, proving the engine talks to a real site end to end. It is not CI and not a
// product — a maintainer runs it locally with a token in the environment (never on the
// command line or in a file that is committed). It cleans up after itself, so it is safe to
// run against a shared automation space.
//
//   LORE_MASTER_ENGINE_BIN=... \
//   CONFLUENCE_DC_BASE_URL=https://atlassian.jato.com/confluence \
//   CONFLUENCE_DC_PAT=<token> CONFLUENCE_DC_SPACE=AUT \
//   node tools/dc-sync-check/check.mjs
import { spawn } from 'node:child_process'
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createMessageConnection, StreamMessageReader, StreamMessageWriter } from 'vscode-jsonrpc/node'

function required (name) {
  const value = process.env[name]
  if (!value) {
    console.error(`${name} is required`)
    process.exit(2)
  }

  return value
}

const baseUrl = required('CONFLUENCE_DC_BASE_URL')
const token = required('CONFLUENCE_DC_PAT')
const enginePath = required('LORE_MASTER_ENGINE_BIN')
const space = process.env.CONFLUENCE_DC_SPACE || 'AUT'
const titlePrefix = process.env.CONFLUENCE_DC_TITLE_PREFIX || 'LoreMaster check'
// A unique H1 so the page never clashes with anything already in the space.
const stamp = new Date().toISOString().replaceAll(/[:.]/g, '-')
const h1 = `Sync check ${stamp}`

const engine = spawn(enginePath, [], { stdio: ['pipe', 'pipe', 'inherit'] })
const rpc = createMessageConnection(new StreamMessageReader(engine.stdout), new StreamMessageWriter(engine.stdin))
rpc.listen()

const workspace = mkdtempSync(join(tmpdir(), 'lore-dc-check-'))
writeFileSync(join(workspace, 'README.md'), `# ${h1}\n\nCreated by tools/dc-sync-check; it should be pruned automatically.\n`)

function settings (parentPageId) {
  return {
    workspaceRoot: workspace,
    settings:      {
      version: 1,
      outputs: [{
        platform:       'confluence',
        baseUrl,
        space,
        parentPageId,
        titlePrefix,
        direction:      'to-platform',
        content:        [{ type: 'markdown', roots: ['.'], excludes: [], template: 'default' }],
        mermaidMode:    'image',
        titleCollision: 'fail',
        linkMode:       'title',
      }],
    },
  }
}

async function main () {
  const session = await rpc.sendRequest('session/open', { baseUrl, edition: 'datacenter', credential: { kind: 'pat', token } })
  console.log(`Connected to ${session.baseUrl} (${session.edition}) as ${session.user.displayName}.`)

  let parentPageId = process.env.CONFLUENCE_DC_PARENT
  if (!parentPageId) {
    const { spaces } = await rpc.sendRequest('space/list', { sessionId: session.sessionId })
    const target = spaces.find(each => each.key === space)
    if (!target || !target.homepageId) {
      throw new Error(`space ${space} not found or has no home page; set CONFLUENCE_DC_PARENT to a page id`)
    }
    parentPageId = target.homepageId
    console.log(`Using the ${space} home page (${parentPageId}) as the parent.`)
  }

  await rpc.sendRequest('settings/save', settings(parentPageId))

  const plan = await rpc.sendRequest('sync/plan', { sessionId: session.sessionId, workspaceRoot: workspace, output: 0 })
  if ((plan.errors ?? []).length > 0 || (plan.counts.conflict ?? 0) > 0) {
    throw new Error(`plan has errors/conflicts: ${JSON.stringify(plan.errors)} conflicts=${plan.counts.conflict ?? 0}`)
  }
  const report = await rpc.sendRequest('sync/execute', { planId: plan.planId, force: false })
  const written = report.pages.filter(page => page.outcome === 'written')
  console.log(`Created ${written.length} page(s):`)
  for (const page of written) {
    console.log(`  ${page.title} -> ${page.url}`)
  }
  if (written.length === 0) {
    throw new Error('nothing was created; the sync did not write the expected page')
  }

  // Clean up: drop the file so the page is an orphan, then prune it to the trash.
  rmSync(join(workspace, 'README.md'))
  const prunePlan = await rpc.sendRequest('sync/plan', { sessionId: session.sessionId, workspaceRoot: workspace, output: 0 })
  const pruneReport = await rpc.sendRequest('sync/execute', { planId: prunePlan.planId, force: false, prune: true })
  const trashed = pruneReport.pages.filter(page => page.outcome === 'trashed')
  console.log(`Pruned ${trashed.length} page(s); the space is left clean.`)

  await rpc.sendRequest('session/close', { sessionId: session.sessionId })
}

try {
  await main()
  console.log('DC sync check passed.')
  cleanup(0)
} catch (error) {
  console.error('DC sync check failed:', error.message ?? error)
  cleanup(1)
}

function cleanup (code) {
  try {
    rmSync(workspace, { recursive: true, force: true })
  } catch {
    // best effort
  }
  rpc.dispose()
  engine.kill()
  process.exit(code)
}
