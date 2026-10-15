import type { JournalSummary, SyncRollbackResult } from '../engine-protocol'
import { describeJournal, leftAlone, previewLines, resultLines } from './rollback-report.mapper'

const journal: JournalSummary = { id: 'j1', startedAt: '2026-10-15T12:00:00Z', baseUrl: 'https://x.atlassian.net/wiki', space: 'AUT', created: 3, updated: 1, trashed: 0, files: 4 }

function result (overrides: Partial<SyncRollbackResult>): SyncRollbackResult {
  return { journalId: 'j1', startedAt: journal.startedAt, pages: [], files: [], complete: true, ...overrides }
}

describe('describeJournal', () => {
  it('says what the sync changed, leaving out what it did not', () => {
    const text = describeJournal(journal, 'en-GB')

    expect(text).toContain('3 pages created')
    expect(text).toContain('1 page updated')
    expect(text).toContain('4 files')
    expect(text).not.toContain('trashed')
  })
})

describe('previewLines', () => {
  it('summarises what the dry run would do', () => {
    const lines = previewLines(result({
      pages: [
        { pageId: '1', title: 'A', kind: 'created', result: 'would-trash' },
        { pageId: '2', title: 'B', kind: 'created', result: 'would-trash' },
        { pageId: '3', title: 'C', kind: 'updated', result: 'would-restore' },
      ],
      files: [{ path: 'a.md', result: 'would-restore' }],
    }))

    expect(lines).toEqual([
      'Delete 2 pages the sync created.',
      'Put 1 updated page back as it were.',
      'Restore 1 local file, the sync state included.',
    ])
  })

  it('warns about what is left alone and what cannot be undone', () => {
    const lines = previewLines(result({
      pages: [
        { pageId: '1', title: 'A', kind: 'created', result: 'would-trash' },
        { pageId: '2', title: 'B', kind: 'updated', result: 'skipped', reason: 'edited' },
        { pageId: '3', title: 'C', kind: 'trashed', result: 'irreversible' },
      ],
      files: [{ path: 'x.md', result: 'skipped' }],
    }))

    expect(lines).toContain('Leave 2 item(s) alone: they were edited after the sync.')
    expect(lines.some(line => line.includes('cannot be restored here'))).toBe(true)
  })
})

describe('resultLines and leftAlone', () => {
  it('lists every page and file with its reason, and counts the skipped ones', () => {
    const outcome = result({
      complete: false,
      pages:    [{ pageId: '9', title: 'Guide', kind: 'created', result: 'skipped', reason: 'edited after the sync' }],
      files:    [{ path: 'a.md', result: 'restored' }],
    })

    expect(resultLines(outcome)).toEqual([
      'page skipped       Guide (9): edited after the sync',
      'file restored      a.md',
    ])
    expect(leftAlone(outcome)).toBe(1)
  })
})
