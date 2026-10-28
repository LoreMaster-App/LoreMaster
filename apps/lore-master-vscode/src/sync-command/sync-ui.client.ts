import * as vscode from 'vscode'
import type { FieldDescriptor, PlatformDescriptor } from '../engine-protocol'
import type { ConnectionMeta } from '../secret-storage'
import { ROLLBACK_SYNC_COMMAND } from '../sync-rollback'
import { createSyncTargetUI } from '../sync-target'
import type { OutputChoice, SyncUI } from './sync-run.use-case'

/** The VS Code-backed {@link SyncUI}: QuickPicks, a progress notification, the output
 *  channel and a status-bar message. */
export function createSyncUI (output: vscode.OutputChannel): SyncUI {
  return {
    ...createSyncTargetUI(),

    async pickConnection (connections: ConnectionMeta[]) {
      const picked = await vscode.window.showQuickPick(
        connections.map(meta => ({ label: meta.baseUrl, description: meta.displayName, meta })),
        { title: 'LoreMaster: connection', placeHolder: 'Which site?' },
      )

      return picked?.meta
    },

    async noConnections () {
      await vscode.window.showInformationMessage('LoreMaster: no connection yet — run "LoreMaster: Add Connection" first.')
    },

    async pickStorageTypes (platforms: PlatformDescriptor[]) {
      const picked = await vscode.window.showQuickPick(
        platforms.map(platform => ({ label: platform.label, description: platform.description, value: platform.id })),
        { title: 'LoreMaster: where to sync', placeHolder: 'Choose one or more storages', canPickMany: true },
      )

      return picked?.map(item => item.value)
    },

    async pickOutputs (choices: OutputChoice[]) {
      const picked = await vscode.window.showQuickPick(
        choices.map(choice => ({ label: choice.label, description: choice.detail, index: choice.index, picked: true })),
        { title: 'LoreMaster: sync to', placeHolder: 'Choose the storages to sync', canPickMany: true },
      )

      return picked?.map(item => item.index)
    },

    async promptField (field: FieldDescriptor, initial?: string) {
      if (field.type === 'choice') {
        const picked = await vscode.window.showQuickPick(
          (field.choices ?? []).map(choice => ({ label: choice.label, description: choice.help, value: choice.value })),
          { title: `LoreMaster: ${field.label}`, placeHolder: field.help },
        )

        return picked?.value
      }
      const pattern = field.pattern === undefined ? undefined : new RegExp(field.pattern)

      return vscode.window.showInputBox({
        title:         `LoreMaster: ${field.label}`,
        prompt:        field.help,
        placeHolder:   field.placeholder ?? field.default,
        value:         initial ?? '',
        validateInput: value => field.required === true && value.trim() === '' ? `${field.label} is required.` : (pattern && value.trim() !== '' && !pattern.test(value.trim()) ? field.patternHelp : undefined),
      })
    },

    async choose (summary: string, conflicts: number) {
      const picked = await vscode.window.showQuickPick(
        [
          { label: '$(check) Run', value: 'run' },
          ...conflicts > 0 ? [{ label: `$(warning) Run and overwrite ${conflicts} page(s) edited on the platform`, value: 'force' }] : [],
          { label: '$(list-unordered) Show details', value: 'details' },
          { label: '$(x) Cancel', value: 'cancel' },
        ],
        { title: `LoreMaster: ${summary}`, placeHolder: 'Review the plan' },
      )

      return (picked?.value ?? 'cancel') as 'cancel' | 'details' | 'force' | 'run'
    },

    writeDetails (lines: string[]) {
      output.appendLine('Plan:')
      for (const line of lines) {
        output.appendLine(`  ${line}`)
      }
      output.show(true)
    },

    async showErrors (errors: string[]) {
      await vscode.window.showErrorMessage(`LoreMaster: the plan has ${errors.length} error(s) and cannot run. See the LoreMaster output.`)
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

    async error (message: string) {
      await vscode.window.showErrorMessage(`LoreMaster: ${message}`)
    },

    offerRollback (journalId: string) {
      void Promise.resolve(vscode.window.showInformationMessage('LoreMaster: synced. Something in the wrong place? This sync can be undone.', 'Roll back')).then(answer => {
        if (answer === 'Roll back') {
          return vscode.commands.executeCommand(ROLLBACK_SYNC_COMMAND, journalId)
        }

        return
      })
    },
  }
}
