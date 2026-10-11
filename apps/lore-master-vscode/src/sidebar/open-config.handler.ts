import * as vscode from 'vscode'

/** The command id contributed in package.json. */
export const OPEN_CONFIG_COMMAND = 'loreMaster.openConfig'

/** Opens VS Code's Settings editor on LoreMaster's settings, where the configuration lives. */
export async function openConfig (): Promise<void> {
  await vscode.commands.executeCommand('workbench.action.openSettings', '@ext:LoreMaster.loremaster')
}
