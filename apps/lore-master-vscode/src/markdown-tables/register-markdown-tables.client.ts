import * as vscode from 'vscode'
import type { TableOperation } from './edit-table.algorithm'
import { editMarkdownTable, IN_TABLE_CONTEXT, TABLE_COMMANDS, tableFormattingEdits } from './edit-markdown-table.handler'
import { findTable } from './find-table.algorithm'

const settings = (): vscode.WorkspaceConfiguration => vscode.workspace.getConfiguration('loreMaster.markdown')

/** Whether the cursor is in a table of a Markdown file, with table editing on. */
function cursorInTable (): boolean {
  const editor = vscode.window.activeTextEditor
  if (editor?.document.languageId !== 'markdown' || !settings().get<boolean>('tableEditing', true) || editor.selections.length !== 1) {
    return false
  }
  const { line } = editor.selection.active

  return editor.document.lineAt(line).text.includes('|') && findTable(editor.document.getText().split(/\r\n|\r|\n/), line) !== undefined
}

/** Registers the table commands, keeps the context key that says when Tab moves between cells, and lines tables up on save when asked to. */
export function registerMarkdownTables (): vscode.Disposable {
  const update = (): void => {
    void vscode.commands.executeCommand('setContext', IN_TABLE_CONTEXT, cursorInTable())
  }
  update()
  const operations = Object.keys(TABLE_COMMANDS) as TableOperation[]

  return vscode.Disposable.from(
    ...operations.map(operation => vscode.commands.registerCommand(TABLE_COMMANDS[operation], () => editMarkdownTable(operation))),
    vscode.window.onDidChangeTextEditorSelection(update),
    vscode.window.onDidChangeActiveTextEditor(update),
    vscode.workspace.onDidChangeConfiguration(event => {
      if (event.affectsConfiguration('loreMaster.markdown')) {
        update()
      }
    }),
    vscode.workspace.onWillSaveTextDocument(event => {
      if (event.document.languageId === 'markdown' && settings().get<boolean>('formatTablesOnSave', false)) {
        event.waitUntil(Promise.resolve(tableFormattingEdits(event.document)))
      }
    }),
  )
}
