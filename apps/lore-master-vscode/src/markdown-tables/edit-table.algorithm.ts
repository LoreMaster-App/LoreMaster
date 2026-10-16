import type { EditResult, TextPosition } from '../text-edits'
import { findTable, scanCells } from './find-table.algorithm'
import { formatTable } from './format-table.algorithm'
import { columnCount, type MarkdownTable, type TableCell } from './markdown-table.model'

/** What a command can do to the table the cursor is in. */
export type TableOperation = 'format' | 'nextCell' | 'previousCell' | 'insertRowAbove' | 'insertRowBelow' | 'deleteRow' | 'insertColumnLeft' | 'insertColumnRight' | 'deleteColumn'

/** The cell the cursor is in: the header counts as row -1, and so does the delimiter row below it. */
function cellAt (table: MarkdownTable, lines: readonly string[], cursor: TextPosition): TableCell {
  const spans = scanCells(lines[cursor.line])
  let column = spans.findIndex(span => cursor.character <= span.end)
  if (column < 0) {
    column = spans.length - 1
  }

  return { row: Math.max(cursor.line - table.startLine - 2, -1), column: Math.min(Math.max(column, 0), columnCount(table) - 1) }
}

const blank = (columns: number): string[] => Array.from({ length: columns }, () => '')

interface Outcome {
  table:  MarkdownTable
  target: TableCell
}

function insertColumn (table: MarkdownTable, at: number): MarkdownTable {
  const put = <T>(values: T[], value: T): T[] => [...values.slice(0, at), value, ...values.slice(at)]

  return { ...table, aligns: put(table.aligns, 'default'), header: put(table.header, ''), rows: table.rows.map(row => put(row, '')) }
}

function deleteColumn (table: MarkdownTable, at: number): MarkdownTable {
  const without = <T>(values: T[]): T[] => values.filter((_, index) => index !== at)

  return { ...table, aligns: without(table.aligns), header: without(table.header), rows: table.rows.map(row => without(row)) }
}

function withRows (table: MarkdownTable, rows: string[][]): MarkdownTable {
  return { ...table, rows }
}

function moveToNext (table: MarkdownTable, cell: TableCell): Outcome {
  const columns = columnCount(table)
  if (cell.column + 1 < columns) {
    return { table, target: { row: cell.row, column: cell.column + 1 } }
  }
  if (cell.row + 1 < table.rows.length) {
    return { table, target: { row: cell.row + 1, column: 0 } }
  }

  // Tab out of the last cell adds a row, so a table can be filled in without leaving the keyboard.
  return { table: withRows(table, [...table.rows, blank(columns)]), target: { row: table.rows.length, column: 0 } }
}

function moveToPrevious (table: MarkdownTable, cell: TableCell): Outcome {
  if (cell.column > 0) {
    return { table, target: { row: cell.row, column: cell.column - 1 } }
  }

  return cell.row >= 0 ? { table, target: { row: cell.row - 1, column: columnCount(table) - 1 } } : { table, target: cell }
}

function operate (table: MarkdownTable, cell: TableCell, operation: TableOperation): Outcome | undefined {
  const columns = columnCount(table)
  switch (operation) {
    case 'format': {
      return { table, target: cell }
    }
    case 'nextCell': {
      return moveToNext(table, cell)
    }
    case 'previousCell': {
      return moveToPrevious(table, cell)
    }
    case 'insertRowAbove': {
      const at = Math.max(cell.row, 0)

      return { table: withRows(table, [...table.rows.slice(0, at), blank(columns), ...table.rows.slice(at)]), target: { row: at, column: cell.column } }
    }
    case 'insertRowBelow': {
      const at = cell.row + 1

      return { table: withRows(table, [...table.rows.slice(0, at), blank(columns), ...table.rows.slice(at)]), target: { row: at, column: cell.column } }
    }
    case 'deleteRow': {
      if (cell.row < 0) {
        return undefined
      }
      const rows = table.rows.filter((_, index) => index !== cell.row)

      return { table: withRows(table, rows), target: { row: Math.min(cell.row, rows.length - 1), column: cell.column } }
    }
    case 'insertColumnLeft': {
      return { table: insertColumn(table, cell.column), target: cell }
    }
    case 'insertColumnRight': {
      return { table: insertColumn(table, cell.column + 1), target: { row: cell.row, column: cell.column + 1 } }
    }
    case 'deleteColumn': {
      return columns > 1 ? { table: deleteColumn(table, cell.column), target: { row: cell.row, column: Math.min(cell.column, columns - 2) } } : undefined
    }
  }
}

/** Where the cursor goes to be in a cell of a formatted table: at the start of its text. */
function caretIn (formatted: readonly string[], table: MarkdownTable, cell: TableCell): TextPosition {
  const index = cell.row < 0 ? 0 : cell.row + 2
  const line = formatted[index]
  const span = scanCells(line)[cell.column]
  const text = line.slice(span.start, span.end)

  // An empty cell has the cursor just inside its left edge, not out at the pipe.
  return { line: table.startLine + index, character: span.start + (text.trim() === '' ? 1 : text.length - text.trimStart().length) }
}

/**
 * Does one operation to the table the cursor is in, and writes the table out lined up with the cursor in the cell the
 * operation leads to. Returns nothing when the cursor is not in a table or the operation cannot be done there.
 */
export function editTable (lines: readonly string[], cursor: TextPosition, operation: TableOperation): EditResult | undefined {
  const table = findTable(lines, cursor.line)
  if (!table) {
    return undefined
  }
  const outcome = operate(table, cellAt(table, lines, cursor), operation)
  if (!outcome) {
    return undefined
  }

  const formatted = formatTable(outcome.table)
  const caret = caretIn(formatted, table, outcome.target)

  return {
    edits:     [{ range: { start: { line: table.startLine, character: 0 }, end: { line: table.endLine, character: lines[table.endLine].length } }, text: formatted.join('\n') }],
    selection: { anchor: caret, active: caret },
  }
}
