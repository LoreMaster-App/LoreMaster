import * as vscode from 'vscode'
import type { Output } from '../engine-protocol'
import type { ConnectionMeta } from '../secret-storage'
import type { StorageType } from '../storage-setup'
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
        { title: 'Lore Master: connection', placeHolder: 'Which site?' },
      )

      return picked?.meta
    },

    async noConnections () {
      await vscode.window.showInformationMessage('Lore Master: no connection yet — run "Lore Master: Add Connection" first.')
    },

    async pickStorageTypes () {
      const picked = await vscode.window.showQuickPick(
        [
          { label: 'Confluence', description: 'Sync Markdown to a Confluence space', value: 'confluence' as StorageType, picked: true },
          { label: 'GitHub Pages', description: 'Publish Markdown as a static site on a gh-pages branch', value: 'github-pages' as StorageType },
        ],
        { title: 'Lore Master: where to sync', placeHolder: 'Choose one or more storages', canPickMany: true },
      )

      return picked?.map(item => item.value)
    },

    async pickOutputs (choices: OutputChoice[]) {
      const picked = await vscode.window.showQuickPick(
        choices.map(choice => ({ label: outputLabel(choice.output), description: outputDescription(choice.output), index: choice.index, picked: true })),
        { title: 'Lore Master: sync to', placeHolder: 'Choose the storages to sync', canPickMany: true },
      )

      return picked?.map(item => item.index)
    },

    promptRepo () {
      return Promise.resolve(vscode.window.showInputBox({
        title:       'Lore Master: GitHub Pages repository',
        prompt:      "owner/name or a clone URL — leave blank to use this repository's origin",
        placeHolder: "(this repository's origin)",
      }))
    },

    promptBranch () {
      return Promise.resolve(vscode.window.showInputBox({ title: 'Lore Master: GitHub Pages branch', prompt: 'Branch to publish to', value: 'gh-pages' }))
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

/** A short, human label for an output in the "sync to…" picker. */
function outputLabel (output: Output): string {
  return output.platform === 'github-pages' ? `GitHub Pages — ${output.branch || 'gh-pages'}` : `Confluence — ${output.space}`
}

/** The second line in the "sync to…" picker: where the output goes. */
function outputDescription (output: Output): string {
  if (output.platform === 'github-pages') {
    return output.repo || "this repository's origin"
  }

  return output.baseUrl
}
