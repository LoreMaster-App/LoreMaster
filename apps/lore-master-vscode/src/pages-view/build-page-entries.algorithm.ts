import type { TreeNode } from '../engine-protocol'
import { fileNameOf } from './page-label.policy'
import { combineStatus } from './page-status.policy'
import type { LabelMode, PageEntry, RemoteCheck } from './page-tree.contract'

const byName = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

/**
 * Nests the engine's flat, parents-first nodes into the hierarchy the sync would build.
 * A node whose parent is not in the list goes at the top, under the configured parent
 * page. Siblings keep the engine's order (numeric prefix, then title) in title mode and
 * are sorted by file name, numbers read as numbers, in file name mode.
 */
export function buildPageEntries (nodes: TreeNode[], mode: LabelMode, remote?: RemoteCheck): PageEntry[] {
  const byPath = new Map<string, PageEntry>()
  const roots: PageEntry[] = []

  for (const node of nodes) {
    const entry: PageEntry = {
      node,
      status:        combineStatus(node.status, remote?.actions[node.path]),
      remoteChecked: remote?.actions[node.path] !== undefined,
      children:      [],
    }
    byPath.set(node.path, entry)
    const parent = node.parent === undefined ? undefined : byPath.get(node.parent)
    if (parent) {
      parent.children.push(entry)
    } else {
      roots.push(entry)
    }
  }

  if (mode === 'fileName') {
    sortByFileName(roots)
  }

  return roots
}

function sortByFileName (entries: PageEntry[]): void {
  entries.sort((a, b) => byName.compare(fileNameOf(a.node), fileNameOf(b.node)))
  for (const entry of entries) {
    sortByFileName(entry.children)
  }
}
