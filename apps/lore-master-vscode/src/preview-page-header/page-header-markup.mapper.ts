import type { TreeStatus } from '../engine-protocol'
import type { PageHeader } from './page-header.contract'

const STATUS_WORDS: Record<TreeStatus, string> = {
  'new':           'Not synced yet',
  'synced':        'Up to date',
  'local-changes': 'Local changes',
}

const escapeHtml = (text: string): string => text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;')

/** The strip at the top of the preview: for each storage that syncs the file, the page title and where the page stands. */
export function pageHeaderMarkup (headers: PageHeader[]): string {
  const rows = headers.map(header => {
    const chip = header.status === undefined
      ? ''
      : ` <span class="loremaster-chip loremaster-chip-${header.status}">${STATUS_WORDS[header.status]}</span>`

    return `<div class="loremaster-page-header-row"><span class="loremaster-page-header-storage">${escapeHtml(header.storage)}</span> ` +
      `<span class="loremaster-page-header-title">${escapeHtml(header.pageTitle)}</span>${chip}</div>`
  })

  return `<div class="loremaster-page-header">${rows.join('')}</div>\n`
}
