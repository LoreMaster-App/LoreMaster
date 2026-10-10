import * as vscode from 'vscode'
import type { Output } from '../engine-protocol'
import { storageLabel } from '../sidebar'
import type { SelectionScope } from './selection-lists.algorithm'

/** The command ids contributed in package.json. */
export const EXCLUDE_FROM_SYNC_COMMAND = 'loreMaster.excludeFromSync'
export const INCLUDE_IN_SYNC_COMMAND = 'loreMaster.includeInSync'

/** A file or folder a view row stands for, and the storage whose view it was chosen in (-1 for Local). */
export interface SelectionTarget {
  path:     string
  isFolder: boolean
  index:    number
}

interface Choice extends vscode.QuickPickItem {
  indexes?: number[]
  all?:     boolean
}

/** Asks which storages an exclusion is for; undefined when the user cancels. */
export async function askExcludeScope (target: SelectionTarget, outputs: { index: number; output: Output }[]): Promise<SelectionScope | undefined> {
  const items: Choice[] = [
    { label: 'All storages', description: 'never synced anywhere', all: true },
    ...outputs.map(each => ({
      label: storageLabel(each.output), indexes: [each.index], picked: each.index === target.index,
    })),
  ]
  const picked = await vscode.window.showQuickPick<Choice>(items, {
    title:       `LoreMaster: do not sync ${target.path}${target.isFolder ? '/' : ''}`,
    placeHolder: 'Leave it out of which storages?',
    canPickMany: true,
  })
  if (!picked || picked.length === 0) {
    return undefined
  }
  if (picked.some(each => each.all)) {
    return { kind: 'all' }
  }

  return { kind: 'storages', indexes: picked.flatMap(each => each.indexes ?? []) }
}

/** Asks which storages a file is brought into; undefined when the user cancels. */
export async function askIncludeIndexes (target: SelectionTarget, outputs: { index: number; output: Output }[]): Promise<number[] | undefined> {
  const items: Choice[] = outputs.map(each => ({
    label: storageLabel(each.output), indexes: [each.index], picked: each.index === target.index,
  }))
  const picked = await vscode.window.showQuickPick<Choice>(items, {
    title:       `LoreMaster: sync ${target.path}`,
    placeHolder: 'Bring it into which storages?',
    canPickMany: true,
  })
  if (!picked || picked.length === 0) {
    return undefined
  }

  return picked.flatMap(each => each.indexes ?? [])
}
