import type { EditResult, TextPosition, TextSelection } from './text-edit.contract'
import { comparePositions, toPosition } from './text-offsets.algorithm'

/**
 * An edit that replaces the text between two offsets, with the cursor given as offsets into the text
 * as it will be after the edit.
 */
export function replaceText (lines: readonly string[], from: number, to: number, replacement: string, anchor: number, active: number): EditResult {
  const text = lines.join('\n')
  const nextLines = (text.slice(0, from) + replacement + text.slice(to)).split('\n')

  return {
    edits:     [{ range: { start: toPosition(lines, from), end: toPosition(lines, to) }, text: replacement }],
    selection: { anchor: toPosition(nextLines, anchor), active: toPosition(nextLines, active) },
  }
}

/** An edit that replaces whole lines, with the cursor given as it will be after the edit. */
export function replaceLines (lines: readonly string[], fromLine: number, toLine: number, replacement: readonly string[], selection: TextSelection): EditResult {
  return {
    edits: [{ range: { start: { line: fromLine, character: 0 }, end: { line: toLine, character: lines[toLine].length } }, text: replacement.join('\n') }],
    selection,
  }
}

/**
 * Keeps a cursor at the same place in its text when lines change length but not number: a cursor after
 * a prefix that was added stays after it, and one inside a prefix that was removed goes to the start.
 */
export function followLines (selection: TextSelection, before: readonly string[], after: readonly string[]): TextSelection {
  const follow = (position: TextPosition): TextPosition => {
    const length = after[position.line]?.length ?? 0
    const delta = length - (before[position.line]?.length ?? 0)

    return { line: position.line, character: Math.min(Math.max(position.character + delta, 0), length) }
  }

  return { anchor: follow(selection.anchor), active: follow(selection.active) }
}

/** The first and last line a selection touches; a last line it only just enters at column 0 is not counted. */
export function linesOf (selection: TextSelection): { from: number, to: number } {
  const [first, last] = comparePositions(selection.anchor, selection.active) <= 0 ? [selection.anchor, selection.active] : [selection.active, selection.anchor]
  const entersOnly = last.character === 0 && last.line > first.line

  return { from: first.line, to: entersOnly ? last.line - 1 : last.line }
}
