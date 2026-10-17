import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { isSiteOutput } from '../config-description'
import type { ConnectionStore } from '../secret-storage'
import { checkRemote } from './check-remote.use-case'
import { checkPages } from './check-pages.use-case'
import type { PagesNode, PagesViewProvider } from './pages-view.client'

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
export async function checkRemoteCommand (deps: CheckRemoteCommandDeps, provider: PagesViewProvider, row?: PagesNode): Promise<void> {
  const { folder, outputs, description } = await provider.configuredOutputs()
  const checkable = description ? outputs.filter(each => description.platforms.some(platform => platform.id === each.output.platform)) : []
  if (!description || checkable.length === 0) {
    await vscode.window.showInformationMessage('LoreMaster: there is no storage to check. Add one with "Add sync storage".')

    return
  }

  let chosen = checkable.find(each => row?.kind === 'storage' && each.index === row.index) ?? checkable[0]
  if (checkable.length > 1 && row?.kind !== 'storage') {
    const picked = await vscode.window.showQuickPick(
      checkable.map(each => ({ label: each.label, description: each.detail, each })),
      { title: 'LoreMaster: check which storage?' },
    ) as { each: typeof chosen } | undefined
    if (!picked) {
      return
    }
    chosen = picked.each
  }

  if (isSiteOutput(description, chosen.output)) {
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
