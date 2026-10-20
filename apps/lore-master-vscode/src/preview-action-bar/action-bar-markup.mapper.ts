import type { TreeStatus } from '../engine-protocol'
import { exportLink } from './action-link.mapper'
import type { PageHeader, PageStatus } from './action-bar.contract'

const STATUS_WORDS: Record<TreeStatus, string> = {
  'new':           'Not synced yet',
  'synced':        'Up to date',
  'local-changes': 'Local changes',
}

/** The switches of the bar: the key the page script stores, and what it is called. */
export const TOGGLES: readonly { key: string, label: string, title: string }[] = [
  { key: 'status', label: 'Sync status', title: 'Show the sync status' },
  { key: 'callouts', label: 'Callouts', title: 'Draw > [!NOTE] and the other alerts as panels' },
  { key: 'diagrams', label: 'Diagram tools', title: 'Show the enlarge and copy buttons on diagrams' },
]

const escapeHtml = (text: string): string => text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;')

function pageRow (page: PageHeader): string {
  const chip = page.status === undefined
    ? ''
    : ` <span class="loremaster-chip loremaster-chip-${page.status}">${STATUS_WORDS[page.status]}</span>`

  return `<span class="loremaster-bar-page"><span class="loremaster-bar-storage">${escapeHtml(page.storage)}</span> ` +
    `<span class="loremaster-bar-title">${escapeHtml(page.pageTitle)}</span>${chip}</span>`
}

function statusMarkup (status: PageStatus): string {
  switch (status.state) {
    case 'checking': {
      return '<span class="loremaster-bar-note">Checking the sync status...</span>'
    }
    case 'untracked': {
      return '<span class="loremaster-bar-note">No storage syncs this file</span>'
    }
    case 'unavailable': {
      return `<span class="loremaster-bar-note">Sync status unavailable: ${escapeHtml(status.reason)}</span>`
    }
    case 'tracked': {
      return status.pages.map(page => pageRow(page)).join('')
    }
  }
}

/**
 * The action bar at the top of the preview: the sync status, an Export button, and a switch for each of LoreMaster's additions
 * to the preview. The switches are driven by the preview script (`action-bar.js`), which keeps their state in the page.
 */
export function actionBarMarkup (document: string, status: PageStatus): string {
  const toggles = TOGGLES.map(toggle =>
    `<button type="button" class="loremaster-bar-toggle" data-toggle="${toggle.key}" aria-pressed="true" title="${escapeHtml(toggle.title)}">${escapeHtml(toggle.label)}</button>`).join('')

  return '<div class="loremaster-action-bar">' +
    `<div class="loremaster-bar-status">${statusMarkup(status)}</div>` +
    `<div class="loremaster-bar-actions"><a class="loremaster-bar-button" href="${escapeHtml(exportLink(document))}" title="Export this file to other formats">Export...</a>` +
    `<span class="loremaster-bar-toggles">${toggles}</span></div>` +
    '</div>\n'
}
