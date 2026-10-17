import * as vscode from 'vscode'
import type { TableOperation } from './edit-table.algorithm'
import { editMarkdownTable, IN_TABLE_CONTEXT, TABLE_COMMANDS, tableFormattingEdits } from './edit-markdown-table.handler'
import { findTable } from './find-table.algorithm'

const settings = (): vscode.WorkspaceConfiguration => vscode.workspace.getConfiguration('loreMaster.markdown')

/** The lines of the run of lines with a pipe around a line, and where the line is in them: all a table can be made of. */
function blockAround (document: vscode.TextDocument, line: number): { lines: string[], index: number } {
  let first = line
  while (first > 0 && document.lineAt(first - 1).text.includes('|')) {
    first -= 1
  }
  let last = line
  while (last < document.lineCount - 1 && document.lineAt(last + 1).text.includes('|')) {
    last += 1
  }
  const lines: string[] = []
  for (let index = first; index <= last; index += 1) {
    lines.push(document.lineAt(index).text)
  }

  return { lines, index: line - first }
}

/** Whether the cursor is in a table of a Markdown file, with table editing on. Looks only at the lines around the cursor. */
function cursorInTable (): boolean {
  const editor = vscode.window.activeTextEditor
  if (editor?.document.languageId !== 'markdown' || !settings().get<boolean>('tableEditing', true) || editor.selections.length !== 1) {
    return false
  }
  const { line } = editor.selection.active
  if (!editor.document.lineAt(line).text.includes('|')) {
    return false
  }
  const { lines, index } = blockAround(editor.document, line)

  return findTable(lines, index) !== undefined
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
