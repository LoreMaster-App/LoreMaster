import type { DisplayOptions } from './page-tree.contract'

/**
 * The commands on the rows of the Sync view's locations (Local and each storage), as contributed in package.json. Each of the
 * display ones sets one choice for the row it was invoked on; the menus offer the other choice for what the row shows now.
 */
export const DISPLAY_COMMANDS: Record<string, Partial<DisplayOptions>> = {
  'loreMaster.showPageTitles': { label: 'title' },
  'loreMaster.showFileNames':  { label: 'fileName' },
  'loreMaster.viewTree':       { view: 'storage' },
  'loreMaster.viewFlat':       { view: 'flat' },
  'loreMaster.viewRepository': { view: 'repo' },
  'loreMaster.showIgnored':    { excluded: 'faded' },
  'loreMaster.hideIgnored':    { excluded: 'hidden' },
}

export const REFRESH_STORAGE_COMMAND = 'loreMaster.refreshStorage'
export const SYNC_STORAGE_COMMAND = 'loreMaster.syncStorage'
export const GENERATE_AND_SYNC_STORAGE_COMMAND = 'loreMaster.generateAndSyncStorage'
