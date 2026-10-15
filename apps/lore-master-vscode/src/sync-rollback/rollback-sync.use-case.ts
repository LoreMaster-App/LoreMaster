import type { EngineClient } from '../engine-process'
import {
  type JournalSummary,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  type SessionOpenResult,
  SYNC_JOURNALS_METHOD,
  SYNC_ROLLBACK_METHOD,
  type SyncJournalsResult,
  type SyncRollbackResult,
} from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { describeJournal, leftAlone, previewLines, resultLines } from './rollback-report.mapper'

/** What the rollback asks of the editor. */
export interface RollbackUI {
  pickJournal (journals: JournalSummary[]): Promise<JournalSummary | undefined>
  /** Shows what would happen and asks; true goes ahead. */
  confirm (title: string, lines: string[]): Promise<boolean>
  /** Offers to discard edits made since the sync; true goes ahead. */
  confirmForce (message: string): Promise<boolean>
  report (lines: string[]): void
  info (message: string): void
  error (message: string): Promise<void>
}

export interface RollbackDeps {
  engine:        EngineClient
  connections:   ConnectionStore
  workspaceRoot: string
  ui:            RollbackUI
  /** The sync to undo; absent lets the user choose (the latest when there is one). */
  journalId?:    string
}

const sameSite = (a: string, b: string): boolean => a.replace(/\/+$/, '').toLowerCase() === b.replace(/\/+$/, '').toLowerCase()

/**
 * Undoes a sync: choose it, preview what would change, confirm, roll back, and say what was
 * left alone. Pages and files edited after the sync are skipped; the user may then choose to
 * discard those edits too. The session is always closed.
 */
export async function rollbackSync (deps: RollbackDeps): Promise<void> {
  const { engine, connections, workspaceRoot, ui } = deps

  const list = await engine.request<SyncJournalsResult>(SYNC_JOURNALS_METHOD, { workspaceRoot })
  if (list.journals.length === 0) {
    ui.info('LoreMaster: no sync to roll back in this workspace yet.')

    return
  }
  const journal = deps.journalId
    ? list.journals.find(each => each.id === deps.journalId)
    : (list.journals.length === 1 ? list.journals[0] : await ui.pickJournal(list.journals))
  if (!journal) {
    return
  }

  const meta = connections.list().find(connection => sameSite(connection.baseUrl, journal.baseUrl))
  const credential = meta ? await connections.credential(meta.baseUrl) : undefined
  if (!meta || !credential) {
    await ui.error(`There is no connection for ${journal.baseUrl}. Add it, then roll back again.`)

    return
  }

  const session = await engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl: meta.baseUrl, edition: meta.edition, credential })
  try {
    await run(deps, journal, session.sessionId)
  } finally {
    try {
      await engine.request(SESSION_CLOSE_METHOD, { sessionId: session.sessionId })
    } catch {
      // Closing is best-effort.
    }
  }
}

async function run (deps: RollbackDeps, journal: JournalSummary, sessionId: string): Promise<void> {
  const { engine, workspaceRoot, ui } = deps
  const base = { sessionId, workspaceRoot, journalId: journal.id }

  const preview = await engine.request<SyncRollbackResult>(SYNC_ROLLBACK_METHOD, { ...base, dryRun: true })
  const lines = previewLines(preview)
  if (lines.length === 0) {
    ui.report(resultLines(preview))
    ui.info('LoreMaster: nothing of that sync can be rolled back.')

    return
  }
  if (!await ui.confirm(`Roll back the sync of ${describeJournal(journal)}?`, lines)) {
    return
  }

  const done = await engine.request<SyncRollbackResult>(SYNC_ROLLBACK_METHOD, base)
  ui.report(resultLines(done))
  const skipped = leftAlone(done)
  if (done.complete) {
    ui.info('LoreMaster: the sync was rolled back.')

    return
  }
  if (skipped > 0 && await ui.confirmForce(`${skipped} page(s) or file(s) were edited after the sync and were left alone. Roll them back too, discarding those edits?`)) {
    const forced = await engine.request<SyncRollbackResult>(SYNC_ROLLBACK_METHOD, { ...base, force: true })
    ui.report(resultLines(forced))
    ui.info(forced.complete ? 'LoreMaster: the sync was rolled back.' : 'LoreMaster: rolled back what could be; see the LoreMaster output for the rest.')

    return
  }
  ui.info('LoreMaster: rolled back what could be; see the LoreMaster output for what was left.')
}
