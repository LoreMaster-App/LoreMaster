import { existsSync } from 'node:fs'
import type { LinkTargets } from './link-check.contract'

/** Answers the link rule from the disk and from what the action bar already knows about each file. */
export function workspaceLinkTargets (isSynced: (path: string) => boolean | undefined): LinkTargets {
  return { exists: path => existsSync(path), isSynced }
}
