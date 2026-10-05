import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { pickWorkspaceFolder } from '../workspace-files'
import { createPagesUI } from './pages-ui.client'
import { publishPages } from './publish-pages.use-case'

/** The command id contributed in package.json. */
export const PUBLISH_PAGES_COMMAND = 'loreMaster.publishPages'

/** What the GitHub Pages command is given at registration. */
export interface PagesCommandDeps {
  engine: EngineClient
  output: vscode.OutputChannel
}

/** Publishes the chosen workspace folder to GitHub Pages. */
export async function publishPagesCommand (deps: PagesCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to publish.')

    return
  }

  await publishPages({ engine: deps.engine, workspaceRoot: folder, ui: createPagesUI(deps.output) })
}
