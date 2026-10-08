import type { TreeNode } from '../engine-protocol'
import type { LabelMode } from './page-tree.contract'

/** The file's name without its folders. */
export function fileNameOf (node: TreeNode): string {
  return node.path.slice(node.path.lastIndexOf('/') + 1)
}

/** What the row is called: the content title, or the file name, as chosen. */
export function pageLabel (node: TreeNode, mode: LabelMode): string {
  return mode === 'title' ? node.title : fileNameOf(node)
}

/** The other name, beside the label, so nothing is hidden by the choice. */
export function pageDescription (node: TreeNode, mode: LabelMode): string {
  return mode === 'title' ? fileNameOf(node) : node.title
}

/** The mode a stored setting value means; anything unknown is the default. */
export function labelModeFrom (value: unknown): LabelMode {
  return value === 'fileName' ? 'fileName' : 'title'
}

/** The other mode, for the toggle. */
export function otherLabelMode (mode: LabelMode): LabelMode {
  return mode === 'title' ? 'fileName' : 'title'
}
