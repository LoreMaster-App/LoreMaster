/** How the Pages view arranges a storage's files. */
export type ViewMode = 'storage' | 'flat' | 'repo'

/** What the view does with files a storage leaves out. */
export type ExcludedMode = 'faded' | 'hidden'

const VIEW_MODES: Set<ViewMode> = new Set(['storage', 'flat', 'repo'])

/** The mode a stored setting value means; anything unknown is the default, the storage tree. */
export function viewModeFrom (value: unknown): ViewMode {
  return VIEW_MODES.has(value as ViewMode) ? value as ViewMode : 'storage'
}

/** The mode a stored setting value means; anything unknown is the default, faded. */
export function excludedModeFrom (value: unknown): ExcludedMode {
  return value === 'hidden' ? 'hidden' : 'faded'
}

/** The other mode, for the toggle. */
export function otherExcludedMode (mode: ExcludedMode): ExcludedMode {
  return mode === 'faded' ? 'hidden' : 'faded'
}

/** What each view mode is called in the picker, and what it shows. */
export const VIEW_MODE_CHOICES: { mode: ViewMode; label: string; description: string }[] = [
  { mode: 'storage', label: 'Storage tree', description: 'the pages as the storage will nest them' },
  { mode: 'flat', label: 'Flat list', description: 'every page in one list' },
  { mode: 'repo', label: 'Repository tree', description: 'the files where they sit in the repository' },
]
