import * as vscode from 'vscode'
import { SYNC_WORKSPACE_COMMAND, syncWorkspace } from './sync-command'

export function activate (context: vscode.ExtensionContext): void {
  context.subscriptions.push(vscode.commands.registerCommand(SYNC_WORKSPACE_COMMAND, syncWorkspace))
}

export function deactivate (): void {}
