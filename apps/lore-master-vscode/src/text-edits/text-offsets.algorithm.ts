import type { TextPosition } from './text-edit.contract'

/** The offset of a position in the text made by joining the lines with newlines. */
export function toOffset (lines: readonly string[], position: TextPosition): number {
  let offset = 0
  const last = Math.min(position.line, lines.length)
  for (let line = 0; line < last; line += 1) {
    offset += lines[line].length + 1
  }

  return offset + position.character
}

/** The position of an offset in the text made by joining the lines with newlines. */
export function toPosition (lines: readonly string[], offset: number): TextPosition {
  let remaining = offset
  for (const [line, text] of lines.entries()) {
    if (remaining <= text.length) {
      return { line, character: remaining }
    }
    remaining -= text.length + 1
  }
  const lastLine = Math.max(lines.length - 1, 0)

  return { line: lastLine, character: lines.at(lastLine)?.length ?? 0 }
}

/** Orders two positions: negative when the first comes first. */
export function comparePositions (left: TextPosition, right: TextPosition): number {
  return left.line === right.line ? left.character - right.character : left.line - right.line
}
