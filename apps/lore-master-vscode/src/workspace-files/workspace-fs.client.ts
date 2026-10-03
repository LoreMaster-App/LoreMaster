import * as vscode from 'vscode'

/**
 * The workspace folder to act on: the only one when there is one, a QuickPick when there
 * are several, and `undefined` when there are none (no folder is open) or the pick was
 * cancelled. The engine owns reading and writing `.lore-master.yaml`; the editor only
 * tells it which folder.
 */
export async function pickWorkspaceFolder (): Promise<string | undefined> {
  const folders = vscode.workspace.workspaceFolders ?? []
  if (folders.length === 0) {
    return undefined
  }
  if (folders.length === 1) {
    return folders[0].uri.fsPath
  }

  const picked = await vscode.window.showQuickPick(
    folders.map(folder => ({ label: folder.name, description: folder.uri.fsPath })),
    { placeHolder: 'Which workspace folder?' },
  )

  return picked?.description
}

/** The folder containing the active editor's file, for a current-file command; undefined
 *  when no editor is active or its file is outside the workspace. */
export function folderForActiveEditor (): string | undefined {
  const editor = vscode.window.activeTextEditor
  if (!editor) {
    return undefined
  }

  return vscode.workspace.getWorkspaceFolder(editor.document.uri)?.uri.fsPath
}
