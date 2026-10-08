import * as vscode from 'vscode'
import { labelModeFrom, otherLabelMode } from './page-label.policy'

/** The command id contributed in package.json. */
export const TOGGLE_PAGE_LABEL_COMMAND = 'loreMaster.togglePageLabel'

/** The setting that holds the choice, so it survives a restart. */
export const PAGE_LABEL_SETTING = 'pages.label'

/** Switches the Pages view between naming pages by content title and by file name. */
export async function togglePageLabel (): Promise<void> {
  const configuration = vscode.workspace.getConfiguration('loreMaster')
  const current = labelModeFrom(configuration.get(PAGE_LABEL_SETTING))

  await configuration.update(PAGE_LABEL_SETTING, otherLabelMode(current), vscode.ConfigurationTarget.Global)
}
