import type { Output } from '../engine-protocol'

/** The first of `base`, `base-2`, `base-3`… that no generator writes to yet. */
export function freeOutputFolder (base: string, taken: readonly string[]): string {
  const used = new Set(taken.map(folder => folder.replace(/\/+$/, '')))
  let candidate = base
  for (let n = 2; used.has(candidate); n++) {
    candidate = `${base}-${n}`
  }

  return candidate
}

/** The folders the storages read: every Markdown content entry's roots, without repeats. */
export function syncedRoots (outputs: readonly Output[]): string[] {
  const all = outputs.flatMap(output => output.content.filter(content => content.type === 'markdown').flatMap(content => content.roots))

  return all.filter((root, position) => all.indexOf(root) === position)
}

/**
 * Whether pages written to `folder` would be read by a storage: some root is the folder, a
 * parent of it, or the whole workspace. A generator whose output is outside every root writes
 * pages that never sync.
 */
export function isFolderSynced (folder: string, outputs: readonly Output[]): boolean {
  const target = folder.replace(/^\.\//, '').replace(/\/+$/, '')

  return syncedRoots(outputs).some(root => {
    const clean = root.replace(/^\.\//, '').replace(/\/+$/, '')

    return clean === '.' || clean === '' || target === clean || target.startsWith(`${clean}/`)
  })
}

/** Why a folder cannot be a generator's output, or undefined when it can. */
export function outputProblem (folder: string): string | undefined {
  const clean = folder.trim().replaceAll('\\', '/')
  if (['', '.', './'].includes(clean)) {
    return 'Name a folder; the pages cannot be written into the workspace root.'
  }
  if (clean === '..' || clean.startsWith('/') || clean.startsWith('../') || clean.includes('/../') || /^[a-z]:/i.test(clean)) {
    return 'The folder must be inside the workspace, written relative to it.'
  }

  return undefined
}
