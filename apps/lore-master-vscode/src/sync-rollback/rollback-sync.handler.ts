import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { JournalSummary } from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { pickWorkspaceFolder } from '../workspace-files'
import { describeJournal } from './rollback-report.mapper'
import { rollbackSync, type RollbackUI } from './rollback-sync.use-case'

/** The command id contributed in package.json. */
export const ROLLBACK_SYNC_COMMAND = 'loreMaster.rollbackSync'

export interface RollbackCommandDeps {
  engine:      EngineClient
  connections: ConnectionStore
  output:      vscode.OutputChannel
}

const ROLL_BACK = 'Roll back'
const FORCE = 'Roll back and discard edits'

function createUI (output: vscode.OutputChannel): RollbackUI {
  return {
    async pickJournal (journals: JournalSummary[]) {
      const picked = await vscode.window.showQuickPick(
        journals.map(journal => ({ label: describeJournal(journal), description: journal.space, detail: journal.baseUrl, journal })),
        { title: 'LoreMaster: which sync should be rolled back?' },
      )

      return picked?.journal
    },
    async confirm (title, lines) {
      return await vscode.window.showWarningMessage(title, { modal: true, detail: lines.join('\n') }, ROLL_BACK) === ROLL_BACK
    },
    async confirmForce (message) {
      return await vscode.window.showWarningMessage(message, { modal: true }, FORCE) === FORCE
    },
    report (lines) {
      for (const line of lines) {
        output.appendLine(line)
      }
      output.show(true)
    },
    info (message) {
      void vscode.window.showInformationMessage(message)
    },
    async error (message) {
      await vscode.window.showErrorMessage(`LoreMaster: ${message}`)
    },
  }
}

/** Undoes a sync: the pages it created are deleted, the ones it updated are put back, and the files get their annotations back. */
export async function rollbackSyncCommand (deps: RollbackCommandDeps, journalId?: string): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder first.')

    return
  }
  try {
    await rollbackSync({ engine: deps.engine, connections: deps.connections, workspaceRoot: folder, ui: createUI(deps.output), journalId })
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}
