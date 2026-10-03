#!/usr/bin/env node
// Dogfood: sync this repo's docs/ to Confluence from CI (#67). Not a product — a Node
// script that drives the built engine over JSON-RPC, using credentials from the
// environment (the GitHub Actions secrets). It fails the job on a plan error or a
// conflict, then executes and prints the report as the job summary. Annotations are not
// committed back: it uses adopt (title-lookup) semantics, so docs/ stays annotation-free.
import { spawn } from 'node:child_process'
import { appendFileSync } from 'node:fs'
import { createMessageConnection, StreamMessageReader, StreamMessageWriter } from 'vscode-jsonrpc/node'

const baseUrl = process.env.CONFLUENCE_BASE_URL
if (!baseUrl) {
  console.log('No CONFLUENCE_BASE_URL — skipping the dogfood sync.')
  process.exit(0)
}
const email = required('CONFLUENCE_EMAIL')
const token = required('CONFLUENCE_API_TOKEN')
const enginePath = required('LORE_MASTER_ENGINE_BIN')
const space = process.env.CONFLUENCE_SPACE || 'LORE'
const parentTitle = process.env.CONFLUENCE_PARENT_TITLE || 'Lore Master'
const titlePrefix = process.env.CONFLUENCE_TITLE_PREFIX || 'Lore Master'
const workspaceRoot = process.cwd()

function required (name) {
  const value = process.env[name]
  if (!value) {
    console.error(`${name} is required`)
    process.exit(1)
  }

  return value
}

function summary (markdown) {
  const file = process.env.GITHUB_STEP_SUMMARY
  if (file) {
    appendFileSync(file, `${markdown}\n`)
  }
  console.log(markdown)
}

function planCounts (plan) {
  const parts = Object.entries(plan.counts).filter(([, n]) => n > 0).map(([kind, n]) => `${n} ${kind}`)

  return parts.length > 0 ? parts.join(', ') : 'nothing to do'
}

const engine = spawn(enginePath, [], { stdio: ['pipe', 'pipe', 'inherit'] })
const connection = createMessageConnection(new StreamMessageReader(engine.stdout), new StreamMessageWriter(engine.stdin))
connection.onNotification('host/progress', progress => {
  if (progress?.total) {
    console.log(`  ${progress.message} (${progress.done}/${progress.total})`)
  }
})
connection.listen()

// A missing space or parent means the tenant isn't set up yet — that is a skip, not a
// failure, so main stays green until it is. Only a real sync problem (a plan error, a
// conflict, a failed page) fails the job.
class NotReadyError extends Error {}

const exitCode = await runDogfood()
connection.dispose()
engine.kill()
process.exit(exitCode)

async function runDogfood () {
  let session
  try {
    session = await connection.sendRequest('session/open', { baseUrl, credential: { kind: 'apitoken', email, token } })
    console.log(`Connected to ${session.baseUrl} (${session.edition}) as ${session.user.displayName} [${session.user.accountId ?? session.user.username ?? '?'}].`)
    await syncDocs(session)

    return 0
  } catch (error) {
    if (error instanceof NotReadyError) {
      summary(`## Lore Master dogfood\n\nSkipped — ${error.message}`)

      return 0
    }
    summary(`\n**Dogfood failed:** ${error.message}`)

    return 1
  } finally {
    if (session) {
      await closeQuietly(session.sessionId)
    }
  }
}

async function syncDocs (session) {
  // space/list can omit a space that is still readable directly, so treat it as a hint,
  // not a gate: log what it returns, then read the space itself via page/search.
  const spaces = await connection.sendRequest('space/list', { sessionId: session.sessionId })
  console.log(`Visible spaces: [${spaces.spaces.map(each => each.key).join(', ')}]`)

  let parentPageId = process.env.CONFLUENCE_PARENT_PAGE_ID
  if (!parentPageId) {
    let found
    try {
      found = await connection.sendRequest('page/search', { sessionId: session.sessionId, spaceKey: space, query: parentTitle })
    } catch (error) {
      throw new NotReadyError(`could not read space "${space}" — is the key right and the token account a member? (${error.message})`)
    }
    const parent = found.pages.find(page => page.title === parentTitle)
    if (!parent) {
      throw new NotReadyError(`parent page "${parentTitle}" not found in ${space} yet — create it, or set CONFLUENCE_PARENT_PAGE_ID`)
    }
    parentPageId = parent.id
  }

  await connection.sendRequest('settings/save', {
    workspaceRoot,
    settings: {
      version: 1,
      outputs: [{
        platform:       'confluence',
        baseUrl:        session.baseUrl,
        space,
        parentPageId,
        titlePrefix,
        direction:      'to-platform',
        content:        [{ type: 'markdown', roots: ['docs'], excludes: [], template: 'default' }],
        mermaidMode:    'image',
        titleCollision: process.env.CONFLUENCE_TITLE_COLLISION || 'adopt',
        linkMode:       'title',
      }],
    },
  })

  const plan = await connection.sendRequest('sync/plan', { sessionId: session.sessionId, workspaceRoot, output: 0 })
  summary(`## Lore Master dogfood\n\n**Plan:** ${planCounts(plan)}`)
  const warnings = plan.warnings ?? []
  for (const warning of warnings) {
    summary(`- warning: ${warning}`)
  }
  const errors = plan.errors ?? []
  const conflicts = plan.counts.conflict ?? 0
  if (errors.length > 0 || conflicts > 0) {
    for (const error of errors) {
      summary(`- error: ${error}`)
    }
    throw new Error(`the plan has ${errors.length} error(s) and ${conflicts} conflict(s); not syncing`)
  }

  const result = await connection.sendRequest('sync/execute', { planId: plan.planId, force: false })
  summary(`\n**Result:**\n${result.pages.map(page => `- ${page.outcome}: ${page.title}${page.error ? ` — ${page.error}` : ''}`).join('\n')}`)
  const failed = result.pages.filter(page => page.outcome === 'failed')
  if (failed.length > 0) {
    throw new Error(`${failed.length} page(s) failed to sync`)
  }
}

async function closeQuietly (sessionId) {
  try {
    await connection.sendRequest('session/close', { sessionId })
  } catch {
    // Closing is best-effort.
  }
}
