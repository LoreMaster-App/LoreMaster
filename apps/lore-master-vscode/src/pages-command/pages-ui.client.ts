import * as vscode from 'vscode'
import type { PagesUI } from './publish-pages.use-case'

/** The VS Code-backed {@link PagesUI}: a progress notification, the output channel, a
 *  status-bar message and error dialogs. */
export function createPagesUI (output: vscode.OutputChannel): PagesUI {
  return {
    withProgress (title, task) {
      return Promise.resolve(vscode.window.withProgress({ location: vscode.ProgressLocation.Notification, title }, () => task()))
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
  }
}
