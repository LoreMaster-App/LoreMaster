import type { DisplayOptions } from './page-tree.contract'

const VIEWS: Record<DisplayOptions['view'], string> = { storage: 'tree', flat: 'flat list', repo: 'repository' }
const LABELS: Record<DisplayOptions['label'], string> = { title: 'page titles', fileName: 'file names' }
const EXCLUDED: Record<DisplayOptions['excluded'], string> = { faded: 'ignored faded', hidden: 'ignored hidden' }

/**
 * What a location row shows now, in words: the arrangement, how pages are named and what happens to ignored files. The buttons on the
 * row say what a click will do (their icon and tooltip name the other choice); this says where it stands, so neither has to be guessed.
 */
export function rowState (options: DisplayOptions): string {
  return `${VIEWS[options.view]} · ${LABELS[options.label]} · ${EXCLUDED[options.excluded]}`
}

/** The same, one choice to a line, for a tooltip. */
export function rowStateLines (options: DisplayOptions, ignoredMeans: string): string[] {
  return [
    `Arranged as: ${VIEWS[options.view]}`,
    `Pages named by: ${LABELS[options.label]}`,
    `${ignoredMeans}: ${options.excluded === 'faded' ? 'shown, faded' : 'hidden'}`,
  ]
}
