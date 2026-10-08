import * as vscode from 'vscode'
import type { FileStore } from './create-agent.use-case'

/**
 * The workspace folder's files through VS Code's file system API, so it works in remote and
 * virtual workspaces too. Folders are created as needed; a missing file reads as undefined.
 */
export function createWorkspaceFileStore (folder: string): FileStore {
  const root = vscode.Uri.file(folder)
  const at = (relativePath: string): vscode.Uri => vscode.Uri.joinPath(root, ...relativePath.split('/'))

  return {
    async read (relativePath) {
      try {
        return new TextDecoder().decode(await vscode.workspace.fs.readFile(at(relativePath)))
      } catch (error) {
        if ((error as { code?: string }).code === 'FileNotFound') {
          return
        }
        throw error
      }
    },
    async write (relativePath, content) {
      const parts = relativePath.split('/')
      if (parts.length > 1) {
        await vscode.workspace.fs.createDirectory(vscode.Uri.joinPath(root, ...parts.slice(0, -1)))
      }
      await vscode.workspace.fs.writeFile(at(relativePath), new TextEncoder().encode(content))
    },
  }
}
