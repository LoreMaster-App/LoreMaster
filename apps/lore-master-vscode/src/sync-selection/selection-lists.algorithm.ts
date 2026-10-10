import type { Output, Settings } from '../engine-protocol'

/** Which storages an exclusion applies to: all of them (the top-level ignore list) or some. */
export type SelectionScope = { kind: 'all' } | { kind: 'storages'; indexes: number[] }

/** A file, or a folder (its entry ends in "/"), as a gitignore-syntax entry. */
export function selectionEntry (path: string, isFolder: boolean): string {
  const clean = path.replace(/\/+$/, '')

  return isFolder ? `${clean}/` : clean
}

function withEntry (list: string[] | undefined, entry: string): string[] {
  const current = list ?? []

  return current.includes(entry) ? current : [...current, entry]
}

function withoutEntry (list: string[] | undefined, entry: string): string[] | undefined {
  if (!list?.includes(entry)) {
    return list
  }
  const rest = list.filter(each => each !== entry)

  return rest.length > 0 ? rest : undefined
}

function withList<K extends 'include' | 'exclude'> (output: Output, key: K, list: string[] | undefined): Output {
  const next = { ...output }
  if (list === undefined) {
    delete next[key]
  } else {
    next[key] = list
  }

  return next
}

/**
 * Leaves `entry` out of the sync: in every storage (the top-level ignore list) or in the
 * chosen ones (their exclude). An entry the storage had in its include is taken out of it,
 * so the two lists never name the same thing.
 */
export function excludeEntry (settings: Settings, entry: string, scope: SelectionScope): Settings {
  if (scope.kind === 'all') {
    return { ...settings, ignore: withEntry(settings.ignore, entry) }
  }
  const outputs = settings.outputs.map((output, index) => {
    if (!scope.indexes.includes(index)) {
      return output
    }

    return withList(withList(output, 'include', withoutEntry(output.include, entry)), 'exclude', withEntry(output.exclude, entry))
  })

  return { ...settings, outputs }
}

/**
 * Brings `entry` into the chosen storages. Where the storage itself excludes exactly that
 * entry, the exclusion is removed; otherwise the entry goes in its include, which beats a
 * broader exclusion (a folder, or the top-level ignore list) by being the more specific.
 */
export function includeEntry (settings: Settings, entry: string, indexes: number[]): Settings {
  const outputs = settings.outputs.map((output, index) => {
    if (!indexes.includes(index)) {
      return output
    }
    if (output.exclude?.includes(entry)) {
      return withList(output, 'exclude', withoutEntry(output.exclude, entry))
    }

    return withList(output, 'include', withEntry(output.include, entry))
  })

  return { ...settings, outputs }
}
