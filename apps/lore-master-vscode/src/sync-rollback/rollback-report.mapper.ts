import type { JournalSummary, SyncRollbackResult } from '../engine-protocol'

const plural = (count: number, word: string): string => `${count} ${word}${count === 1 ? '' : 's'}`

/** One line describing a sync that can be rolled back, for the picker. */
export function describeJournal (journal: JournalSummary, locale?: string): string {
  const when = new Date(journal.startedAt).toLocaleString(locale)
  const parts = [
    ...journal.created > 0 ? [`${plural(journal.created, 'page')} created`] : [],
    ...journal.updated > 0 ? [`${plural(journal.updated, 'page')} updated`] : [],
    ...journal.trashed > 0 ? [`${plural(journal.trashed, 'page')} trashed`] : [],
    ...journal.files > 0 ? [plural(journal.files, 'file')] : [],
  ]

  return `${when}: ${parts.join(', ') || 'no changes'}`
}

/** What a rollback is about to do, from its dry run: the lines the user confirms. */
export function previewLines (preview: SyncRollbackResult): string[] {
  const count = (results: string[]): number => preview.pages.filter(page => results.includes(page.result)).length
  const lines: string[] = []

  const trash = count(['would-trash'])
  const restore = count(['would-restore'])
  const files = preview.files.filter(file => file.result === 'would-restore').length
  if (trash > 0) {
    lines.push(`Delete ${plural(trash, 'page')} the sync created.`)
  }
  if (restore > 0) {
    lines.push(`Put ${plural(restore, 'updated page')} back as ${restore === 1 ? 'it' : 'they'} were.`)
  }
  if (files > 0) {
    lines.push(`Restore ${plural(files, 'local file')}, annotations included.`)
  }

  const skipped = preview.pages.filter(page => page.result === 'skipped')
  const skippedFiles = preview.files.filter(file => file.result === 'skipped')
  if (skipped.length + skippedFiles.length > 0) {
    lines.push(`Leave ${skipped.length + skippedFiles.length} item(s) alone: they were edited after the sync.`)
  }
  const irreversible = count(['irreversible'])
  if (irreversible > 0) {
    lines.push(`${plural(irreversible, 'page')} the sync trashed cannot be restored here; use the Confluence trash.`)
  }

  return lines
}

/** The detailed log lines of a finished (or previewed) rollback. */
export function resultLines (result: SyncRollbackResult): string[] {
  const lines = [
    ...result.pages.map(page => `page ${page.result.padEnd(13)} ${page.title} (${page.pageId})${page.reason ? `: ${page.reason}` : ''}`),
    ...result.files.map(file => `file ${file.result.padEnd(13)} ${file.path}${file.reason ? `: ${file.reason}` : ''}`),
  ]

  return lines
}

/** Whether anything was left undone, so a forced run might still change something. */
export function leftAlone (result: SyncRollbackResult): number {
  return result.pages.filter(page => page.result === 'skipped').length + result.files.filter(file => file.result === 'skipped').length
}
