import * as vscode from 'vscode'
import { excludedModeFrom, otherExcludedMode, VIEW_MODE_CHOICES, viewModeFrom } from './display-options.policy'

/** The command ids contributed in package.json. */
export const CHOOSE_VIEW_MODE_COMMAND = 'loreMaster.choosePagesView'
export const TOGGLE_EXCLUDED_COMMAND = 'loreMaster.togglePagesExcluded'

/** The settings that hold the choices, so they survive a restart. */
export const PAGES_VIEW_SETTING = 'pages.view'
export const PAGES_EXCLUDED_SETTING = 'pages.excluded'

/** Asks how the Pages view arranges files: storage tree, flat list or repository tree. */
export async function chooseViewMode (): Promise<void> {
  const configuration = vscode.workspace.getConfiguration('loreMaster')
  const current = viewModeFrom(configuration.get(PAGES_VIEW_SETTING))
  const picked = await vscode.window.showQuickPick(
    VIEW_MODE_CHOICES.map(choice => ({ label: choice.label, description: choice.description, picked: choice.mode === current, mode: choice.mode })),
    { title: 'LoreMaster: how to arrange the pages' },
  )
  if (picked) {
    await configuration.update(PAGES_VIEW_SETTING, picked.mode, vscode.ConfigurationTarget.Global)
  }
}

/** Switches between showing left-out files faded in place and hiding them. */
export async function toggleExcluded (): Promise<void> {
  const configuration = vscode.workspace.getConfiguration('loreMaster')
  const current = excludedModeFrom(configuration.get(PAGES_EXCLUDED_SETTING))

  await configuration.update(PAGES_EXCLUDED_SETTING, otherExcludedMode(current), vscode.ConfigurationTarget.Global)
}
