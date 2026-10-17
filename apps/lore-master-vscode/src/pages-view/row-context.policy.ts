import type { DisplayOptions } from './page-tree.contract'

/** The kinds of location row, as the `viewItem` of a menu `when` clause names them. */
export type LocationKind = 'loreMasterLocal' | 'loreMasterStorage'

/**
 * The context value of a location row: its kind and how it is shown now, one word each, so a menu can offer the other choice
 * (`viewItem =~ /label-title/` shows "Show File Names") and a row's icons change with what it shows.
 */
export function rowContext (kind: LocationKind, options: DisplayOptions): string {
  return `${kind} label-${options.label} view-${options.view} ignored-${options.excluded}`
}
