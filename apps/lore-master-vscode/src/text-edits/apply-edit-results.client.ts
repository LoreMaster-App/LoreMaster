import * as vscode from 'vscode'
import { combineEditResults } from './combine-edit-results.algorithm'
import type { EditResult, TextPosition, TextSelection } from './text-edit.contract'
import { comparePositions } from './text-offsets.algorithm'

/** The document's lines, without their line endings. */
export function documentLines (document: vscode.TextDocument): string[] {
  return document.getText().split(/\r\n|\r|\n/)
}

/** The editor's selections, first in the document first. */
export function selectionsOf (editor: vscode.TextEditor): TextSelection[] {
  return editor.selections
    .map(selection => ({ anchor: toPosition(selection.anchor), active: toPosition(selection.active) }))
    .sort((left, right) => comparePositions(left.anchor, right.anchor))
}

/** Applies what each cursor wanted done and moves the cursors to where they should be. */
export async function applyEditResults (editor: vscode.TextEditor, results: readonly EditResult[]): Promise<boolean> {
  const { edits, selections } = combineEditResults(results)
  if (edits.length === 0) {
    return false
  }
  const applied = await editor.edit(builder => {
    for (const edit of edits) {
      builder.replace(new vscode.Range(edit.range.start.line, edit.range.start.character, edit.range.end.line, edit.range.end.character), edit.text)
    }
  })
  if (applied) {
    editor.selections = selections.map(selection => new vscode.Selection(selection.anchor.line, selection.anchor.character, selection.active.line, selection.active.character))
  }

  return applied
}

function toPosition (position: vscode.Position): TextPosition {
  return { line: position.line, character: position.character }
}
