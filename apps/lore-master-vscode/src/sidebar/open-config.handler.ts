import { join } from 'node:path'
import * as vscode from 'vscode'
import { pickWorkspaceFolder } from '../workspace-files'

/** The command id contributed in package.json. */
export const OPEN_CONFIG_COMMAND = 'loreMaster.openConfig'

/** Opens the workspace's .lore-master.yaml in the editor, or says how to create it. */
export async function openConfig (): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder first.')

    return
  }

  const uri = vscode.Uri.file(join(folder, '.lore-master.yaml'))
  try {
    const document = await vscode.workspace.openTextDocument(uri)
    await vscode.window.showTextDocument(document)
  } catch {
    await vscode.window.showInformationMessage('LoreMaster: no .lore-master.yaml yet — run Sync or "Add sync storage" to create it.')
  }
}
