import type { EditResult, TextEdit, TextPosition, TextSelection } from './text-edit.contract'
import { comparePositions } from './text-offsets.algorithm'

/**
 * Puts the results of several cursors together. Each result was worked out against the document as it is,
 * so the edits can be applied as they are, but where a cursor ends up has to allow for the edits of the
 * cursors above it. Results whose edits would overlap an earlier one are left out: two cursors in the same
 * word cannot both change it.
 */
export function combineEditResults (results: readonly EditResult[]): { edits: TextEdit[], selections: TextSelection[] } {
  const ordered = results
    .filter(result => result.edits.length > 0)
    .sort((left, right) => comparePositions(left.edits[0].range.start, right.edits[0].range.start))

  const accepted: EditResult[] = []
  for (const result of ordered) {
    const clashes = accepted.some(earlier => earlier.edits.some(edit => result.edits.some(next => comparePositions(next.range.start, edit.range.end) < 0)))
    if (!clashes) {
      accepted.push(result)
    }
  }

  const selections = accepted.map((result, index) => {
    let selection = result.selection
    for (const earlier of accepted.slice(0, index)) {
      for (const edit of earlier.edits) {
        selection = { anchor: shiftPast(selection.anchor, edit), active: shiftPast(selection.active, edit) }
      }
    }

    return selection
  })

  return { edits: accepted.flatMap(result => result.edits), selections }
}

/** Where a position that comes after an edit ends up once the edit is made. */
function shiftPast (position: TextPosition, edit: TextEdit): TextPosition {
  if (comparePositions(position, edit.range.end) < 0) {
    return position
  }
  const inserted = edit.text.split('\n')
  const lineDelta = inserted.length - 1 - (edit.range.end.line - edit.range.start.line)
  if (position.line !== edit.range.end.line) {
    return { line: position.line + lineDelta, character: position.character }
  }
  const lastLength = inserted.at(-1)?.length ?? 0
  const base = inserted.length === 1 ? edit.range.start.character + lastLength : lastLength

  return { line: position.line + lineDelta, character: base + position.character - edit.range.end.character }
}
