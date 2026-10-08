import * as vscode from 'vscode'
import { createSyncUI, type SyncUI } from '../sync-command'

/** A {@link SyncUI} that never asks and never interrupts: it is what a sync started by a file
 *  change runs with. Plans run as planned, progress shows in the status bar, details go to the
 *  output channel without opening it, and a page edited on the platform is left alone rather than
 *  overwritten. What went wrong is kept in `problems`, for the caller to act on. */
export interface SilentSyncUI extends SyncUI {
  readonly problems: string[]
}

export function createSilentSyncUI (output: vscode.OutputChannel): SilentSyncUI {
  const problems: string[] = []
  const log = (line: string): void => { output.appendLine(line) }

  return {
    ...createSyncUI(output),
    problems,

    choose: () => Promise.resolve('run' as const),

    writeDetails (lines: string[]) {
      log('Plan:')
      for (const line of lines) {
        log(`  ${line}`)
      }
    },

    showErrors (errors: string[]) {
      for (const error of errors) {
        problems.push(error)
        log(`error: ${error}`)
      }

      return Promise.resolve()
    },

    withProgress: (title, task) => Promise.resolve(vscode.window.withProgress({ location: vscode.ProgressLocation.Window, title }, progress =>
      task(step => progress.report({ message: step.total > 0 ? `${step.message} (${step.done}/${step.total})` : step.message })))),

    report (lines: string[]) {
      for (const line of lines) {
        log(line)
      }
    },

    status: () => {},

    confirmForce: () => {
      log('Some pages changed on the platform since the last sync, so they were left alone. Sync them yourself to overwrite.')

      return Promise.resolve(false)
    },

    // Setting up storages asks questions; a watch has to start from outputs that are already set up.
    pickStorageTypes: () => Promise.resolve(undefined),

    error (message: string) {
      problems.push(message)
      log(`error: ${message}`)

      return Promise.resolve()
    },
  }
}
