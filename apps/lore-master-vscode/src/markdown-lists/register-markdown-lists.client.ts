import * as vscode from 'vscode'
import { IN_LIST_CONTEXT, LIST_ENTER_COMMAND, LIST_INDENT_COMMAND, LIST_OUTDENT_COMMAND, listEnter, listIndent, listOutdent } from './edit-list.handler'
import { parseListItem } from './parse-list-item.algorithm'

/** Whether every cursor of the active editor is on a list item of a Markdown file, with list editing on. */
function cursorsOnListItems (): boolean {
  const editor = vscode.window.activeTextEditor
  if (editor?.document.languageId !== 'markdown' || !vscode.workspace.getConfiguration('loreMaster.markdown').get<boolean>('listEditing', true)) {
    return false
  }

  return editor.selections.every(selection => parseListItem(editor.document.lineAt(selection.active.line).text) !== undefined)
}

/** Registers the list keys, and keeps the context key that says when they apply up to date. */
export function registerMarkdownLists (): vscode.Disposable {
  const update = (): void => {
    void vscode.commands.executeCommand('setContext', IN_LIST_CONTEXT, cursorsOnListItems())
  }
  update()

  return vscode.Disposable.from(
    vscode.commands.registerCommand(LIST_ENTER_COMMAND, listEnter),
    vscode.commands.registerCommand(LIST_INDENT_COMMAND, listIndent),
    vscode.commands.registerCommand(LIST_OUTDENT_COMMAND, listOutdent),
    vscode.window.onDidChangeTextEditorSelection(update),
    vscode.window.onDidChangeActiveTextEditor(update),
    vscode.workspace.onDidChangeConfiguration(event => {
      if (event.affectsConfiguration('loreMaster.markdown')) {
        update()
      }
    }),
  )
}
