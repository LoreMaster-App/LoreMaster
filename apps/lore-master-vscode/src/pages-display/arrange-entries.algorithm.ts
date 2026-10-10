/** Anything with a workspace-relative path that can be placed in a folder tree. */
export interface Placeable {
  path: string
}

/** A folder of the repository tree, with the folders and files directly in it. */
export interface RepoFolder<T> {
  name:    string
  /** Workspace-relative, without a trailing slash; empty for the root. */
  path:    string
  folders: RepoFolder<T>[]
  files:   T[]
}

const byName = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

function lastSegment (path: string): string {
  return path.slice(path.lastIndexOf('/') + 1)
}

/** Every item in one list, ordered by path. */
export function flatten<T> (items: T[], pathOf: (item: T) => string): T[] {
  return [...items].sort((a, b) => byName.compare(pathOf(a), pathOf(b)))
}

/**
 * Places files where they sit in the repository: folders first, then files, each in name order.
 * A folder with no file below it does not appear, so the tree shows only the Markdown.
 */
export function repoTree<T> (items: T[], pathOf: (item: T) => string): RepoFolder<T> {
  const root: RepoFolder<T> = { name: '', path: '', folders: [], files: [] }
  for (const item of items) {
    const segments = pathOf(item).split('/')
    let folder = root
    for (let depth = 0; depth < segments.length - 1; depth++) {
      const path = segments.slice(0, depth + 1).join('/')
      let child = folder.folders.find(each => each.path === path)
      if (!child) {
        child = { name: segments[depth], path, folders: [], files: [] }
        folder.folders.push(child)
      }
      folder = child
    }
    folder.files.push(item)
  }
  sortFolder(root, pathOf)

  return root
}

function sortFolder<T> (folder: RepoFolder<T>, pathOf: (item: T) => string): void {
  folder.folders.sort((a, b) => byName.compare(a.name, b.name))
  folder.files.sort((a, b) => byName.compare(lastSegment(pathOf(a)), lastSegment(pathOf(b))))
  for (const child of folder.folders) {
    sortFolder(child, pathOf)
  }
}
