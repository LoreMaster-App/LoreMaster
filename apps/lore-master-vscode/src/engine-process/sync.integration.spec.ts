import { execFileSync } from 'node:child_process'
import { existsSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import { type AddressInfo } from 'node:net'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import {
  SESSION_OPEN_METHOD,
  type SessionOpenResult,
  SETTINGS_SAVE_METHOD,
  SYNC_EXECUTE_METHOD,
  type SyncExecuteResult,
  SYNC_PLAN_METHOD,
  type SyncPlanResult,
} from '../engine-protocol'
import { createEngineClient, type EngineClient } from './engine-rpc.client'

// Drives the REAL Go engine through a full sync — session/open, settings/save, plan,
// execute — against an in-process fake Confluence (Data Center v1) with a two-file
// workspace on disk. It proves the shipped engine binary talks to a Confluence-shaped API
// over HTTP and carries a sync end to end: the TS-side counterpart to the Go remotesync
// test (#127, #139). Gated exactly like engine-rpc.integration.spec.ts: it runs when a
// built engine is configured or `go` is on PATH, and is skipped otherwise.

const repoRoot = resolve(__dirname, '../../../..')

function goAvailable (): boolean {
  try {
    execFileSync('go', ['version'], { stdio: 'pipe' })

    return true
  } catch {
    return false
  }
}

function buildEngine (): string {
  const directory = mkdtempSync(join(tmpdir(), 'lore-engine-'))
  const binary = join(directory, process.platform === 'win32' ? 'lore-master-engine.exe' : 'lore-master-engine')
  execFileSync('go', ['build', '-o', binary, './apps/lore-master-engine'], { cwd: repoRoot, stdio: 'pipe' })

  return binary
}

// --- the fake Confluence Data Center -------------------------------------------------

interface FakePage { id: string; title: string; spaceKey: string; parentId: string; version: number; marked: boolean; property?: { value: unknown; version: number } }

function sameTitle (a: string, b: string): boolean {
  return a.toLowerCase() === b.toLowerCase()
}

/** Reads a quoted value after a CQL fragment ("title = ", "ancestor = "), undoing the
 *  backslash escaping the client applies. */
function cqlValue (cql: string, prefix: string): string {
  const index = cql.indexOf(prefix)
  if (index === -1) {
    return ''
  }
  const rest = cql.slice(index + prefix.length)
  if (!rest.startsWith('"')) {
    return ''
  }
  let value = ''
  for (let i = 1; i < rest.length; i++) {
    if (rest[i] === '\\' && i + 1 < rest.length) {
      value += rest[++i]

      continue
    }
    if (rest[i] === '"') {
      break
    }
    value += rest[i]
  }

  return value
}

/** An in-process Confluence Data Center (v1), faithful to the one dialect the engine uses:
 *  the page body is never read — pages are matched by title and parent. `_links.base` is
 *  left empty, so the engine resolves page URLs against the base it already knows. */
function fakeConfluence (): { server: Server; rootId: string; pageCount: () => number } {
  const pages: FakePage[] = []
  const store = (spaceKey: string, parentId: string, title: string): FakePage => {
    const page: FakePage = { id: `p${pages.length + 1}`, title, spaceKey, parentId, version: 1, marked: false }
    pages.push(page)

    return page
  }
  const root = store('ENG', '', 'Engineering Home')
  const find = (id: string): FakePage | undefined => pages.find(page => page.id === id)
  const taken = (spaceKey: string, title: string): boolean => pages.some(page => page.spaceKey === spaceKey && sameTitle(page.title, title))

  const below = (start: string, rootId: string): boolean => {
    let id = start
    for (let hops = 0; hops <= pages.length; hops++) {
      const page = find(id)
      if (!page || page.parentId === '') {
        return false
      }
      if (page.parentId === rootId) {
        return true
      }
      id = page.parentId
    }

    return false
  }
  const wire = (page: FakePage): Record<string, unknown> => ({
    id:      page.id,
    title:   page.title,
    space:   { id: 1, key: page.spaceKey },
    version: { number: page.version },
    ...(page.parentId && { ancestors: [{ id: page.parentId }] }),
    _links:  { webui: `/display/${page.spaceKey}/${page.id}`, base: '' },
  })
  const server = createServer((request, response) => {
    const url = new URL(request.url ?? '', 'http://localhost')
    const path = url.pathname
    const send = (status: number, body: unknown): void => {
      response.writeHead(status, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify(body))
    }
    const list = (results: Record<string, unknown>[]): void => send(200, { results, size: results.length, limit: 250, _links: {} })
    const id = (/^\/rest\/api\/content\/([^/]+)/.exec(path) ?? [])[1] ?? ''

    let raw = ''
    request.on('data', chunk => { raw += chunk })
    request.on('end', () => {
      if (path === '/rest/api/content' && request.method === 'POST') {
        const body = JSON.parse(raw || '{}') as { title: string; space: { key: string }; ancestors?: { id: string }[] }
        if (taken(body.space.key, body.title)) {
          return send(400, { message: `A page with this title already exists: ${body.title}` })
        }
        const created = store(body.space.key, body.ancestors?.at(-1)?.id ?? '', body.title)

        return send(200, wire(created))
      }
      if (path === '/rest/api/content/search' && request.method === 'GET') {
        const cql = url.searchParams.get('cql') ?? ''
        if (cql.includes('label = "lore-master"')) {
          const rootId = cqlValue(cql, 'ancestor = ')

          return list(pages.filter(page => page.marked && below(page.id, rootId)).map(page => wire(page)))
        }
        const key = cqlValue(cql, 'space = ')
        const title = cqlValue(cql, 'title = ')

        return list(pages.filter(page => page.spaceKey === key && sameTitle(page.title, title)).map(page => wire(page)))
      }
      if (/^\/rest\/api\/content\/[^/]+\/child\/page$/.test(path) && request.method === 'GET') {
        return list(pages.filter(page => page.parentId === id).map(page => wire(page)))
      }
      if (/^\/rest\/api\/content\/[^/]+\/label$/.test(path) && request.method === 'POST') {
        const page = find(id)
        if (page) {
          page.marked = true
        }

        return send(200, {})
      }
      // Content property (the source-path marker): read, create, then update on a re-mark.
      if (/^\/rest\/api\/content\/[^/]+\/property\/[^/]+$/.test(path) && request.method === 'GET') {
        const property = find(id)?.property

        return property ? send(200, { value: property.value, version: { number: property.version } }) : send(404, { message: 'No property found' })
      }
      if (/^\/rest\/api\/content\/[^/]+\/property$/.test(path) && request.method === 'POST') {
        const page = find(id)
        if (!page) {
          return send(404, { message: 'No content found' })
        }
        page.property = { value: (JSON.parse(raw || '{}') as { value: unknown }).value, version: 1 }

        return send(200, {})
      }
      if (/^\/rest\/api\/content\/[^/]+\/property\/[^/]+$/.test(path) && request.method === 'PUT') {
        const page = find(id)
        if (!page) {
          return send(404, { message: 'No content found' })
        }
        const body = JSON.parse(raw || '{}') as { value: unknown; version: { number: number } }
        page.property = { value: body.value, version: body.version.number }

        return send(200, {})
      }
      if (/^\/rest\/api\/content\/[^/]+$/.test(path) && request.method === 'PUT') {
        const page = find(id)
        if (!page) {
          return send(404, { message: 'No content found' })
        }
        const body = JSON.parse(raw || '{}') as { title: string; ancestors?: { id: string }[]; version: { number: number } }
        if (body.version.number !== page.version + 1) {
          return send(409, { message: 'Version must be incremented on update' })
        }
        page.title = body.title
        page.parentId = body.ancestors?.at(-1)?.id ?? page.parentId
        page.version = body.version.number

        return send(200, wire(page))
      }
      if (/^\/rest\/api\/content\/[^/]+$/.test(path) && request.method === 'GET') {
        const page = find(id)

        return page ? send(200, wire(page)) : send(404, { message: 'No content found' })
      }
      if (path === '/rest/api/space' && request.method === 'GET') {
        return list([{ id: 1, key: 'ENG', name: 'Engineering', type: 'global', homepage: { id: root.id } }])
      }
      if (path === '/rest/api/user/current' && request.method === 'GET') {
        return request.headers.authorization
          ? send(200, { type: 'known', username: 'ada', userKey: 'k1', displayName: 'Ada Lovelace' })
          : send(401, { message: 'Unauthorized' })
      }

      return send(404, { message: `unexpected ${request.method ?? '?'} ${path}` })
    })
  })

  return { server, rootId: root.id, pageCount: () => pages.length }
}

// --- the test ------------------------------------------------------------------------

const configured = process.env.LORE_MASTER_ENGINE_BIN
const canRun = (configured && existsSync(configured)) || goAvailable()
const describeWithEngine = canRun ? describe : describe.skip

describeWithEngine('a full sync through the real engine against a fake Confluence', () => {
  let binaryPath: string
  let client: EngineClient
  let fake: ReturnType<typeof fakeConfluence>
  let baseUrl: string
  let workspace: string

  beforeAll(async () => {
    binaryPath = configured && existsSync(configured) ? configured : buildEngine()
    fake = fakeConfluence()
    await new Promise<void>(resolve => fake.server.listen(0, '127.0.0.1', resolve))
    baseUrl = `http://127.0.0.1:${(fake.server.address() as AddressInfo).port}`

    workspace = mkdtempSync(join(tmpdir(), 'lore-ws-'))
    writeFileSync(join(workspace, 'README.md'), '# Home\n\nSee [setup](setup.md).\n')
    writeFileSync(join(workspace, 'setup.md'), '# Setup\n')
  }, 180_000)

  afterAll(async () => {
    client?.dispose()
    await new Promise<void>(resolve => fake.server.close(() => resolve()))
  })

  function settingsParams (): unknown {
    return {
      workspaceRoot: workspace,
      settings:      {
        version: 1,
        outputs: [{
          platform:       'confluence',
          baseUrl,
          space:          'ENG',
          parentPageId:   fake.rootId,
          titlePrefix:    'ENG',
          direction:      'to-platform',
          content:        [{ type: 'markdown', roots: ['.'], excludes: [], template: 'default' }],
          mermaidMode:    'image',
          titleCollision: 'adopt',
          linkMode:       'title',
        }],
      },
    }
  }

  it('creates the pages, writes annotations back, and is unchanged on the second run', async () => {
    client = createEngineClient({ binaryPath })

    const session = await client.request<SessionOpenResult>(SESSION_OPEN_METHOD, {
      baseUrl, edition: 'datacenter', credential: { kind: 'pat', token: 'a-token' },
    })
    expect(session.edition).toBe('datacenter')
    expect(session.user.displayName).toBe('Ada Lovelace')

    await client.request(SETTINGS_SAVE_METHOD, settingsParams())

    const plan = await client.request<SyncPlanResult>(SYNC_PLAN_METHOD, { sessionId: session.sessionId, workspaceRoot: workspace, output: 0 })
    expect(plan.errors ?? []).toEqual([])
    expect(plan.counts.create).toBe(2)

    const report = await client.request<SyncExecuteResult>(SYNC_EXECUTE_METHOD, { planId: plan.planId, force: false })
    expect(report.warnings ?? []).toEqual([])
    const titles = report.pages.filter(page => page.outcome === 'written').map(page => page.title)
    expect(titles.sort((a, b) => a.localeCompare(b))).toEqual(['ENG: Home', 'ENG: Setup'])
    expect(fake.pageCount()).toBe(3) // the two created pages plus the seeded parent

    // The engine wrote the sync annotation back into each file, so the second run detects
    // true no-ops rather than re-adopting by title.
    expect(readFileSync(join(workspace, 'README.md'), 'utf8')).toContain('<!-- lore-master')

    const second = await client.request<SyncPlanResult>(SYNC_PLAN_METHOD, { sessionId: session.sessionId, workspaceRoot: workspace, output: 0 })
    expect(second.counts.unchanged).toBe(2)
    expect(fake.pageCount()).toBe(3) // nothing re-created
  }, 30_000)
})
