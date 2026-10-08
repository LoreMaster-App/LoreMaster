import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { ConnectionStore } from '../secret-storage'
import { checkRemote } from './check-remote.use-case'
import type { PagesViewProvider } from './pages-view.client'

/** The command id contributed in package.json. */
export const CHECK_REMOTE_COMMAND = 'loreMaster.checkRemote'

export interface CheckRemoteCommandDeps {
  engine:      EngineClient
  connections: ConnectionStore
}

/**
 * Asks the platform where the Pages view's pages stand (read-only) and shows the answer in
 * the view. With several Confluence storages it asks which; GitHub Pages has no remote
 * state to compare.
 */
export async function checkRemoteCommand (deps: CheckRemoteCommandDeps, provider: PagesViewProvider): Promise<void> {
  const { folder, outputs } = await provider.configuredOutputs()
  const confluence = outputs.filter(each => each.output.platform === 'confluence')
  if (confluence.length === 0) {
    await vscode.window.showInformationMessage('LoreMaster: there is no Confluence storage to check. Add one with "Add sync storage".')

    return
  }

  let chosen = confluence[0]
  if (confluence.length > 1) {
    const picked = await vscode.window.showQuickPick(
      confluence.map(each => ({ label: `Confluence · ${each.output.space}`, description: each.output.baseUrl, each })),
      { title: 'LoreMaster: check which storage?' },
    ) as { each: typeof chosen } | undefined
    if (!picked) {
      return
    }
    chosen = picked.each
  }

  const outcome = await vscode.window.withProgress(
    { location: vscode.ProgressLocation.Window, title: 'LoreMaster: checking the platform' },
    () => checkRemote({ engine: deps.engine, connections: deps.connections, workspaceRoot: folder }, chosen.output, chosen.index),
  )
  if (outcome.ok) {
    provider.setRemote(chosen.index, outcome.check)
  } else {
    await vscode.window.showErrorMessage(`LoreMaster: ${outcome.reason}`)
  }
}
