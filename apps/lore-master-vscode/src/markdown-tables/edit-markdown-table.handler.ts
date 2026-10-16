import * as vscode from 'vscode'
import { applyEditResults, documentLines } from '../text-edits'
import { editTable, type TableOperation } from './edit-table.algorithm'
import { formatAllTables } from './format-table.algorithm'

/** The command ids contributed in package.json, by what they do. */
export const TABLE_COMMANDS: Record<TableOperation, string> = {
  format:            'loreMaster.markdown.formatTable',
  nextCell:          'loreMaster.markdown.tableNextCell',
  previousCell:      'loreMaster.markdown.tablePreviousCell',
  insertRowAbove:    'loreMaster.markdown.tableInsertRowAbove',
  insertRowBelow:    'loreMaster.markdown.tableInsertRowBelow',
  deleteRow:         'loreMaster.markdown.tableDeleteRow',
  insertColumnLeft:  'loreMaster.markdown.tableInsertColumnLeft',
  insertColumnRight: 'loreMaster.markdown.tableInsertColumnRight',
  deleteColumn:      'loreMaster.markdown.tableDeleteColumn',
}

/** The context key that says the cursor is in a table, so Tab moves between its cells there. */
export const IN_TABLE_CONTEXT = 'loreMaster.markdown.inTable'

/** Does one operation to the table at the cursor; Tab and Shift+Tab fall back to their usual job outside a table. */
export async function editMarkdownTable (operation: TableOperation): Promise<void> {
  const editor = vscode.window.activeTextEditor
  if (editor?.document.languageId !== 'markdown') {
    return
  }
  const result = editTable(documentLines(editor.document), { line: editor.selection.active.line, character: editor.selection.active.character }, operation)
  if (result) {
    await applyEditResults(editor, [result])

    return
  }
  if (operation === 'nextCell') {
    await vscode.commands.executeCommand('tab')
  } else if (operation === 'previousCell') {
    await vscode.commands.executeCommand('outdent')
  }
}

/** The edits that line up every table of a document, for a save. */
export function tableFormattingEdits (document: vscode.TextDocument): vscode.TextEdit[] {
  return formatAllTables(documentLines(document)).map(edit => vscode.TextEdit.replace(new vscode.Range(edit.range.start.line, edit.range.start.character, edit.range.end.line, edit.range.end.character), edit.text))
}
