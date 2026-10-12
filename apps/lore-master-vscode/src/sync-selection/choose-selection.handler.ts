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

function describe (targets: SelectionTarget[]): string {
  const [first] = targets

  return first && targets.length === 1 ? `${first.path}${first.isFolder ? '/' : ''}` : `${targets.length} items`
}

function isChosenIn (targets: SelectionTarget[], index: number): boolean {
  return targets.some(target => target.index === index)
}

/**
 * Asks which storages an exclusion is for; undefined when the user cancels. With one storage
 * there is nothing to choose, so it is not asked.
 */
export async function askExcludeScope (targets: SelectionTarget[], outputs: { index: number; output: Output }[]): Promise<SelectionScope | undefined> {
  const [only] = outputs
  if (only && outputs.length === 1) {
    return { kind: 'storages', indexes: [only.index] }
  }
  const items: Choice[] = [
    { label: 'All storages', description: 'never synced anywhere', all: true },
    ...outputs.map(each => ({
      label: storageLabel(each.output), indexes: [each.index], picked: isChosenIn(targets, each.index),
    })),
  ]
  const picked = await vscode.window.showQuickPick<Choice>(items, {
    title:       `LoreMaster: do not sync ${describe(targets)}`,
    placeHolder: 'Leave out of which storages?',
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

/** Asks which storages files are brought into; undefined when the user cancels. Not asked with one storage. */
export async function askIncludeIndexes (targets: SelectionTarget[], outputs: { index: number; output: Output }[]): Promise<number[] | undefined> {
  const [only] = outputs
  if (only && outputs.length === 1) {
    return [only.index]
  }
  const items: Choice[] = outputs.map(each => ({
    label: storageLabel(each.output), indexes: [each.index], picked: isChosenIn(targets, each.index),
  }))
  const picked = await vscode.window.showQuickPick<Choice>(items, {
    title:       `LoreMaster: sync ${describe(targets)}`,
    placeHolder: 'Bring into which storages?',
    canPickMany: true,
  })
  if (!picked || picked.length === 0) {
    return undefined
  }

  return picked.flatMap(each => each.indexes ?? [])
}
