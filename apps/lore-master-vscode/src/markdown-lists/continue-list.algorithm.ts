import type { EditResult, TextEdit, TextPosition } from '../text-edits'
import { indentListItems } from './indent-list.algorithm'
import { markerEnd, parseListItem } from './parse-list-item.algorithm'
import { renumberOrdered } from './renumber-list.algorithm'

/**
 * What Enter does at the cursor in a list item: starts the next item, with the next number and an empty box
 * when it is a task, and carries what was after the cursor down into it. On an item that is empty it ends the
 * list instead (a nested one comes out a level). Returns nothing when the cursor is not on an item, or is in
 * its marker, and Enter should do what it always does.
 */
export function continueList (lines: readonly string[], cursor: TextPosition): EditResult | undefined {
  const line = lines[cursor.line]
  const item = parseListItem(line)
  if (!item || cursor.character < markerEnd(item)) {
    return undefined
  }

  if (item.content.trim() === '' && cursor.character >= line.trimEnd().length) {
    return endList(lines, cursor, item.indent)
  }

  const nextMarker = item.ordered ? `${item.number + 1}${item.delimiter}` : item.marker
  const box = item.checkbox === undefined ? '' : '[ ] '
  const prefix = `${item.prefix}${nextMarker}${item.spacing}${box}`
  const after = line.slice(cursor.character).trimStart()

  const edits: TextEdit[] = [{
    range: { start: { line: cursor.line, character: line.slice(0, cursor.character).trimEnd().length }, end: { line: cursor.line, character: line.length } },
    text:  `\n${prefix}${after}`,
  }]
  if (item.ordered) {
    const next = [...lines.slice(0, cursor.line), line.slice(0, cursor.character), prefix + after, ...lines.slice(cursor.line + 1)]
    const renumbered = renumberOrdered(next, cursor.line + 2)
    for (const [index, text] of renumbered) {
      edits.push({ range: { start: { line: index - 1, character: 0 }, end: { line: index - 1, character: lines[index - 1].length } }, text })
    }
  }
  const caret = { line: cursor.line + 1, character: prefix.length }

  return { edits, selection: { anchor: caret, active: caret } }
}

/** Enter on an empty item: the item goes, or comes out of its parent when it is nested. */
function endList (lines: readonly string[], cursor: TextPosition, indent: number): EditResult | undefined {
  const here = { anchor: cursor, active: cursor }
  if (indent > 0) {
    return indentListItems(lines, here, 'outdent')
  }
  const line = lines[cursor.line]
  const item = parseListItem(line)
  const kept = item ? item.prefix.trimEnd() : ''
  const caret = { line: cursor.line, character: kept.length }

  return {
    edits:     [{ range: { start: { line: cursor.line, character: 0 }, end: { line: cursor.line, character: line.length } }, text: kept }],
    selection: { anchor: caret, active: caret },
  }
}
