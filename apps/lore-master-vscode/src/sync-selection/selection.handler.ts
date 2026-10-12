import * as vscode from 'vscode'
import type { Output } from '../engine-protocol'
import { type ChangeSelectionDeps, excludeFromSync, includeInSync } from './change-selection.use-case'
import { askExcludeScope, askIncludeIndexes, type SelectionTarget } from './choose-selection.handler'
import { selectionEntries } from './selection-lists.algorithm'

export interface SelectionCommandDeps extends ChangeSelectionDeps {
  outputs: () => Promise<{ index: number; output: Output }[]>
  /** Called after the settings changed, so the views re-read. */
  changed: () => void
}

async function report (action: () => Promise<void>, done: () => void): Promise<void> {
  try {
    await action()
    done()
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}

/** Adds the files and folders to the do-not-sync list of the storages the user picks, in one write. */
export async function excludeFromSyncCommand (deps: SelectionCommandDeps, targets: SelectionTarget[]): Promise<void> {
  if (targets.length === 0) {
    return
  }
  const scope = await askExcludeScope(targets, await deps.outputs())
  if (!scope) {
    return
  }

  await report(() => excludeFromSync(deps, selectionEntries(targets), scope), deps.changed)
}

/** Brings left-out files into the storages the user picks, in one write. */
export async function includeInSyncCommand (deps: SelectionCommandDeps, targets: SelectionTarget[]): Promise<void> {
  if (targets.length === 0) {
    return
  }
  const indexes = await askIncludeIndexes(targets, await deps.outputs())
  if (!indexes) {
    return
  }

  await report(() => includeInSync(deps, selectionEntries(targets), indexes), deps.changed)
}
