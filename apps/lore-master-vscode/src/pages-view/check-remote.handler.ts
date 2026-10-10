import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { isGitHubOutput } from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { storageDescription, storageLabel } from '../sidebar'
import { checkRemote } from './check-remote.use-case'
import { checkPages } from './check-pages.use-case'
import type { PagesViewProvider } from './pages-view.client'

/** The command id contributed in package.json. */
export const CHECK_REMOTE_COMMAND = 'loreMaster.checkRemote'

export interface CheckRemoteCommandDeps {
  engine:      EngineClient
  connections: ConnectionStore
}

/**
 * Asks where the Pages view's pages stand, read-only. For a Confluence storage the answer
 * lands in the view; for a GitHub Pages output it is a message saying whether a publish would
 * change anything, because a site has no per-page remote state. With several storages it asks
 * which.
 */
export async function checkRemoteCommand (deps: CheckRemoteCommandDeps, provider: PagesViewProvider): Promise<void> {
  const { folder, outputs } = await provider.configuredOutputs()
  const checkable = outputs.filter(each => each.output.platform === 'confluence' || isGitHubOutput(each.output))
  if (checkable.length === 0) {
    await vscode.window.showInformationMessage('LoreMaster: there is no storage to check. Add one with "Add sync storage".')

    return
  }

  let chosen = checkable[0]
  if (checkable.length > 1) {
    const picked = await vscode.window.showQuickPick(
      checkable.map(each => ({
        label:       isGitHubOutput(each.output) ? storageLabel(each.output) : `Confluence · ${each.output.space}`,
        description: isGitHubOutput(each.output) ? storageDescription(each.output) : each.output.baseUrl,
        each,
      })),
      { title: 'LoreMaster: check which storage?' },
    ) as { each: typeof chosen } | undefined
    if (!picked) {
      return
    }
    chosen = picked.each
  }

  if (isGitHubOutput(chosen.output)) {
    const site = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Window, title: 'LoreMaster: comparing the site with the published branch' },
      () => checkPages(deps.engine, folder, chosen.index),
    )
    if (!site.ok) {
      await vscode.window.showErrorMessage(`LoreMaster: ${site.reason}`)
    } else if (site.upToDate) {
      await vscode.window.showInformationMessage(`LoreMaster: ${site.message}`)
    } else {
      await vscode.window.showWarningMessage(`LoreMaster: ${site.message}`)
    }

    return
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
