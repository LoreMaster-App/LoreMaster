import type { TextEdit } from '../text-edits'
import { findTables } from './find-table.algorithm'
import type { ColumnAlign, MarkdownTable } from './markdown-table.model'

const MINIMUM_WIDTH = 3

/** How many columns of a monospaced editor a text takes: wide characters (CJK, emoji) take two and combining marks none. */
export function displayWidth (text: string): number {
  let width = 0
  for (const character of text) {
    const code = character.codePointAt(0) ?? 0
    if (code === 0x200D || (code >= 0xFE00 && code <= 0xFE0F) || /\p{M}/u.test(character)) {
      continue
    }
    const wide = (code >= 0x1100 && code <= 0x115F) || (code >= 0x2E80 && code <= 0xA4CF) || (code >= 0xAC00 && code <= 0xD7A3) ||
      (code >= 0xF900 && code <= 0xFAFF) || (code >= 0xFE30 && code <= 0xFE6F) || (code >= 0xFF00 && code <= 0xFF60) ||
      (code >= 0xFFE0 && code <= 0xFFE6) || (code >= 0x1F300 && code <= 0x1F64F) || (code >= 0x1F900 && code <= 0x1F9FF) || (code >= 0x20000 && code <= 0x3FFFD)
    width += wide ? 2 : 1
  }

  return width
}

function pad (text: string, width: number, align: ColumnAlign): string {
  const space = width - displayWidth(text)
  switch (align) {
    case 'right': {
      return ' '.repeat(space) + text
    }
    case 'center': {
      const left = Math.floor(space / 2)

      return ' '.repeat(left) + text + ' '.repeat(space - left)
    }
    default: {
      return text + ' '.repeat(space)
    }
  }
}

function delimiter (width: number, align: ColumnAlign): string {
  switch (align) {
    case 'left': {
      return `:${'-'.repeat(width - 1)}`
    }
    case 'right': {
      return `${'-'.repeat(width - 1)}:`
    }
    case 'center': {
      return `:${'-'.repeat(width - 2)}:`
    }
    default: {
      return '-'.repeat(width)
    }
  }
}

/** The lines of a table with its pipes lined up, the delimiter row as wide as its column, and the alignments kept. */
export function formatTable (table: MarkdownTable): string[] {
  const widths = table.aligns.map((_, column) => Math.max(MINIMUM_WIDTH, displayWidth(table.header[column]), ...table.rows.map(row => displayWidth(row[column]))))
  const row = (cells: string[]): string => `${table.indent}| ${cells.map((cell, column) => pad(cell, widths[column], table.aligns[column])).join(' | ')} |`

  return [
    row(table.header),
    `${table.indent}| ${table.aligns.map((align, column) => delimiter(widths[column], align)).join(' | ')} |`,
    ...table.rows.map(cells => row(cells)),
  ]
}

/** The edits that line up every table of a document that is not lined up already. */
export function formatAllTables (lines: readonly string[]): TextEdit[] {
  const edits: TextEdit[] = []
  for (const table of findTables(lines)) {
    const formatted = formatTable(table)
    const original = lines.slice(table.startLine, table.endLine + 1)
    if (formatted.length !== original.length || formatted.some((line, index) => line !== original[index])) {
      edits.push({
        range: { start: { line: table.startLine, character: 0 }, end: { line: table.endLine, character: lines[table.endLine].length } },
        text:  formatted.join('\n'),
      })
    }
  }

  return edits
}
