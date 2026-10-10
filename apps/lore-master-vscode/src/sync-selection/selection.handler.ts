import * as vscode from 'vscode'
import type { Output } from '../engine-protocol'
import { type ChangeSelectionDeps, excludeFromSync, includeInSync } from './change-selection.use-case'
import { askExcludeScope, askIncludeIndexes, type SelectionTarget } from './choose-selection.handler'
import { selectionEntry } from './selection-lists.algorithm'

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

/** Adds the file or folder to the do-not-sync list of the storages the user picks. */
export async function excludeFromSyncCommand (deps: SelectionCommandDeps, target: SelectionTarget | undefined): Promise<void> {
  if (!target) {
    return
  }
  const scope = await askExcludeScope(target, await deps.outputs())
  if (!scope) {
    return
  }

  await report(() => excludeFromSync(deps, selectionEntry(target.path, target.isFolder), scope), deps.changed)
}

/** Brings a left-out file into the storages the user picks. */
export async function includeInSyncCommand (deps: SelectionCommandDeps, target: SelectionTarget | undefined): Promise<void> {
  if (!target) {
    return
  }
  const indexes = await askIncludeIndexes(target, await deps.outputs())
  if (!indexes) {
    return
  }

  await report(() => includeInSync(deps, selectionEntry(target.path, target.isFolder), indexes), deps.changed)
}
