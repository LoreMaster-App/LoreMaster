import type { TreeNode } from '../engine-protocol'
import type { ExcludedMode, ViewMode } from '../pages-display'

/** How a page is named in the tree: by its content title or by its file name. */
export type LabelMode = 'title' | 'fileName'

/** How a location (Local or a storage) is shown: how its pages are named and arranged, and what happens to the files it leaves out. */
export interface DisplayOptions {
  label:    LabelMode
  view:     ViewMode
  excluded: ExcludedMode
}

/** Where a page stands, combining the file's own status with the platform's when known. */
export type PageStatus = 'new' | 'synced' | 'local-changes' | 'remote-changes' | 'conflict' | 'adopt'

/** What a read-only plan said about an output, kept until the next change to the files. */
export interface RemoteCheck {
  /** The plan's action kind per workspace-relative file. */
  actions: Record<string, string>
  /** Pages the sync made whose file is gone. */
  orphans: { title: string; pageId?: string; url?: string }[]
}

/** One page in the tree, with the pages nested under it. */
export interface PageEntry {
  node:          TreeNode
  /** Absent for an output that does not track pages in the files (github-pages). */
  status?:       PageStatus
  /** True when the status was confirmed against the platform. */
  remoteChecked: boolean
  children:      PageEntry[]
}
