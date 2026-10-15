import type { EngineClient } from '../engine-process'
import {
  type JournalSummary,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  SYNC_JOURNALS_METHOD,
  SYNC_ROLLBACK_METHOD,
  type SyncRollbackResult,
} from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { rollbackSync, type RollbackUI } from './rollback-sync.use-case'

const journal: JournalSummary = { id: 'j1', startedAt: '2026-10-15T12:00:00Z', baseUrl: 'https://x.atlassian.net/wiki/', space: 'AUT', created: 2, updated: 0, trashed: 0, files: 2 }

function rollbackResult (overrides: Partial<SyncRollbackResult>): SyncRollbackResult {
  return { journalId: 'j1', startedAt: journal.startedAt, pages: [], files: [], complete: true, ...overrides }
}

interface Calls {
  rollbacks: { dryRun?: boolean, force?: boolean, journalId?: string }[]
  closed:    number
}

function engineWith (journals: JournalSummary[], answer: (params: { dryRun?: boolean, force?: boolean }) => SyncRollbackResult): { engine: EngineClient, calls: Calls } {
  const calls: Calls = { rollbacks: [], closed: 0 }
  const engine = {
    request (method: string, params?: unknown) {
      switch (method) {
        case SYNC_JOURNALS_METHOD: {
          return Promise.resolve({ journals } as never)
        }
        case SESSION_OPEN_METHOD: {
          return Promise.resolve({ sessionId: 's1' } as never)
        }
        case SESSION_CLOSE_METHOD: {
          calls.closed += 1

          return Promise.resolve(null as never)
        }
        case SYNC_ROLLBACK_METHOD: {
          const rollback = params as { dryRun?: boolean, force?: boolean, journalId?: string }
          calls.rollbacks.push(rollback)

          return Promise.resolve(answer(rollback) as never)
        }
        default: {
          return Promise.reject(new Error(`unexpected ${method}`))
        }
      }
    },
  } as unknown as EngineClient

  return { engine, calls }
}

const connections: ConnectionStore = {
  list:       () => [{ baseUrl: 'https://x.atlassian.net/wiki', edition: 'cloud', displayName: 'X', user: 'u' }],
  credential: () => Promise.resolve({ kind: 'basic', email: 'a@b', token: 't' } as never),
  add:        () => Promise.resolve(),
  remove:     () => Promise.resolve(),
}

function uiWith (answers: { confirm?: boolean, force?: boolean } = {}) {
  const log = { reports: [] as string[][], infos: [] as string[], errors: [] as string[], confirmed: [] as string[][], forced: 0 }
  const ui: RollbackUI = {
    pickJournal: list => Promise.resolve(list[0]),
    confirm:     (_title, lines) => {
      log.confirmed.push(lines)

      return Promise.resolve(answers.confirm ?? true)
    },
    confirmForce: () => {
      log.forced += 1

      return Promise.resolve(answers.force ?? false)
    },
    report: lines => { log.reports.push(lines) },
    info:   message => { log.infos.push(message) },
    error:  message => {
      log.errors.push(message)

      return Promise.resolve()
    },
  }

  return { ui, log }
}

const preview = rollbackResult({ dryRun: true, pages: [{ pageId: '1', title: 'A', kind: 'created', result: 'would-trash' }], files: [{ path: 'a.md', result: 'would-restore' }] })

describe('rollbackSync', () => {
  it('previews, confirms, rolls back and closes the session', async () => {
    const { engine, calls } = engineWith([journal], params => params.dryRun ? preview : rollbackResult({ pages: [{ pageId: '1', title: 'A', kind: 'created', result: 'trashed' }] }))
    const { ui, log } = uiWith()

    await rollbackSync({ engine, connections, workspaceRoot: '/w', ui })

    expect(calls.rollbacks.map(call => [call.dryRun ?? false, call.force ?? false])).toEqual([[true, false], [false, false]])
    expect(log.confirmed[0]).toContain('Delete 1 page the sync created.')
    expect(log.infos).toEqual(['LoreMaster: the sync was rolled back.'])
    expect(calls.closed).toBe(1)
  })

  it('changes nothing when the user does not confirm', async () => {
    const { engine, calls } = engineWith([journal], () => preview)
    const { ui } = uiWith({ confirm: false })

    await rollbackSync({ engine, connections, workspaceRoot: '/w', ui })

    expect(calls.rollbacks).toHaveLength(1)
    expect(calls.closed).toBe(1)
  })

  it('offers to discard later edits when pages were left alone, and rolls those back when agreed', async () => {
    const skipped = rollbackResult({ complete: false, pages: [{ pageId: '2', title: 'B', kind: 'created', result: 'skipped', reason: 'edited' }] })
    const { engine, calls } = engineWith([journal], params => params.dryRun ? preview : (params.force ? rollbackResult({}) : skipped))
    const { ui, log } = uiWith({ force: true })

    await rollbackSync({ engine, connections, workspaceRoot: '/w', ui })

    expect(log.forced).toBe(1)
    expect(calls.rollbacks.map(call => call.force ?? false)).toEqual([false, false, true])
    expect(log.infos).toEqual(['LoreMaster: the sync was rolled back.'])
  })

  it('leaves edited pages alone when the user declines to force', async () => {
    const skipped = rollbackResult({ complete: false, pages: [{ pageId: '2', title: 'B', kind: 'created', result: 'skipped' }] })
    const { engine, calls } = engineWith([journal], params => params.dryRun ? preview : skipped)
    const { ui, log } = uiWith({ force: false })

    await rollbackSync({ engine, connections, workspaceRoot: '/w', ui })

    expect(calls.rollbacks).toHaveLength(2)
    expect(log.infos[0]).toContain('rolled back what could be')
  })

  it('says so when there is nothing to roll back, and when there is no connection', async () => {
    const empty = engineWith([], () => preview)
    const first = uiWith()
    await rollbackSync({ engine: empty.engine, connections, workspaceRoot: '/w', ui: first.ui })
    expect(first.log.infos[0]).toContain('no sync to roll back')

    const { engine } = engineWith([{ ...journal, baseUrl: 'https://other.example/wiki' }], () => preview)
    const second = uiWith()
    await rollbackSync({ engine, connections, workspaceRoot: '/w', ui: second.ui })
    expect(second.log.errors[0]).toContain('no connection for https://other.example/wiki')
  })

  it('rolls back the journal it is given', async () => {
    const other = { ...journal, id: 'j2' }
    const { engine, calls } = engineWith([journal, other], params => params.dryRun ? preview : rollbackResult({}))
    const { ui } = uiWith()

    await rollbackSync({ engine, connections, workspaceRoot: '/w', ui, journalId: 'j2' })

    expect(calls.rollbacks.every(call => call.journalId === 'j2')).toBe(true)
  })
})
