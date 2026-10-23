import * as vscode from 'vscode'
import { toggleTaskInLine } from './toggle-task-line.algorithm'

/**
 * Flips one task in a Markdown file: the checkbox of the preview asks for it with the file and the line. Only a task list item of a
 * Markdown document is ever changed, whatever the link says. A file that was not open is saved again, since nobody is looking at
 * its unsaved edit; an open one is left for its author to save.
 */
export async function toggleTask (params: Record<string, string>): Promise<void> {
  const line = params.line === undefined || params.line === '' ? NaN : Number(params.line)
  if (params.document === undefined || !Number.isSafeInteger(line) || line < 0) { return }
  const uri = vscode.Uri.parse(params.document)
  const wasOpen = vscode.workspace.textDocuments.some(each => each.uri.toString() === uri.toString())
  const document = await vscode.workspace.openTextDocument(uri)
  if (document.languageId !== 'markdown' || line >= document.lineCount) { return }

  const text = document.lineAt(line).text
  const flipped = toggleTaskInLine(text)
  if (flipped === undefined) { return }
  const edit = new vscode.WorkspaceEdit()
  edit.replace(uri, new vscode.Range(line, 0, line, text.length), flipped)
  const applied = await vscode.workspace.applyEdit(edit)
  if (applied && !wasOpen) { await document.save() }
}
