import * as vscode from 'vscode'
import type { ConnectionMeta } from '../secret-storage'
import { createSyncTargetUI } from '../sync-target'
import type { SyncUI } from './sync-run.use-case'

/** The VS Code-backed {@link SyncUI}: QuickPicks, a progress notification, the output
 *  channel and a status-bar message. */
export function createSyncUI (output: vscode.OutputChannel): SyncUI {
  return {
    ...createSyncTargetUI(),

    async pickConnection (connections: ConnectionMeta[]) {
      const picked = await vscode.window.showQuickPick(
        connections.map(meta => ({ label: meta.baseUrl, description: meta.displayName, meta })),
        { title: 'Lore Master: connection', placeHolder: 'Which site?' },
      )

      return picked?.meta
    },

    async noConnections () {
      await vscode.window.showInformationMessage('Lore Master: no connection yet — run "Lore Master: Add Connection" first.')
    },

    async choose (summary: string) {
      const picked = await vscode.window.showQuickPick(
        [
          { label: '$(check) Run', value: 'run' },
          { label: '$(list-unordered) Show details', value: 'details' },
          { label: '$(x) Cancel', value: 'cancel' },
        ],
        { title: `Lore Master: ${summary}`, placeHolder: 'Review the plan' },
      )

      return (picked?.value ?? 'cancel') as 'cancel' | 'details' | 'run'
    },

    writeDetails (lines: string[]) {
      output.appendLine('Plan:')
      for (const line of lines) {
        output.appendLine(`  ${line}`)
      }
      output.show(true)
    },

    async showErrors (errors: string[]) {
      await vscode.window.showErrorMessage(`Lore Master: the plan has ${errors.length} error(s) and cannot run. See the Lore Master output.`)
      output.show(true)
    },

    withProgress (title, task) {
      return Promise.resolve(vscode.window.withProgress({ location: vscode.ProgressLocation.Notification, title }, progress =>
        task(step => progress.report({ message: step.total > 0 ? `${step.message} (${step.done}/${step.total})` : step.message }))))
    },

    report (lines: string[]) {
      for (const line of lines) {
        output.appendLine(line)
      }
      output.show(true)
    },

    status (message: string) {
      vscode.window.setStatusBarMessage(message, 5000)
    },

    async confirmForce (message: string) {
      return await vscode.window.showWarningMessage(message, { modal: true }, 'Force') === 'Force'
    },

    async error (message: string) {
      await vscode.window.showErrorMessage(`Lore Master: ${message}`)
    },
  }
}
