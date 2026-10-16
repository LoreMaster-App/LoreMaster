import type { ColumnAlign, MarkdownTable } from './markdown-table.model'

/** Where a cell's text lies between its pipes in a line, from the character after one pipe to the one before the next. */
export interface CellSpan {
  start: number
  end:   number
}

/**
 * The cells of a table line. A leading pipe and a trailing one are not cells, and a pipe written `\|` is text: GitHub splits
 * cells at every other pipe, even one in a code span.
 */
export function scanCells (line: string): CellSpan[] {
  const cells: CellSpan[] = []
  let begin = line.length - line.trimStart().length
  if (line[begin] === '|') {
    begin += 1
  }
  for (let index = begin; index < line.length; index += 1) {
    if (line[index] === '\\') {
      index += 1
    } else if (line[index] === '|') {
      cells.push({ start: begin, end: index })
      begin = index + 1
    }
  }
  if (line.slice(begin).trim() !== '') {
    cells.push({ start: begin, end: line.length })
  }

  return cells
}

/** The trimmed text of each cell of a line. */
export function cellTexts (line: string): string[] {
  return scanCells(line).map(span => line.slice(span.start, span.end).trim())
}

/** Whether a line is the delimiter row of a table: cells of dashes, each with a colon at either end or neither. */
export function isDelimiterRow (line: string): boolean {
  if (!line.includes('|') && !line.includes(':')) {
    return false
  }
  const cells = cellTexts(line)

  return cells.length > 0 && cells.every(cell => /^:?-+:?$/.test(cell))
}

function alignOf (cell: string): ColumnAlign {
  const left = cell.startsWith(':')
  const right = cell.endsWith(':')
  if (left && right) {
    return 'center'
  }
  if (left) {
    return 'left'
  }

  return right ? 'right' : 'default'
}

/** The table that has a line in it, if the line is part of one. */
export function findTable (lines: readonly string[], lineIndex: number): MarkdownTable | undefined {
  if (!lines[lineIndex]?.includes('|')) {
    return undefined
  }
  let first = lineIndex
  while (first > 0 && lines[first - 1].includes('|')) {
    first -= 1
  }
  let last = lineIndex
  while (last < lines.length - 1 && lines[last + 1].includes('|')) {
    last += 1
  }

  for (let delimiter = first + 1; delimiter <= last; delimiter += 1) {
    if (isDelimiterRow(lines[delimiter]) && lineIndex >= delimiter - 1) {
      return buildTable(lines, delimiter - 1, last)
    }
  }

  return undefined
}

function buildTable (lines: readonly string[], startLine: number, endLine: number): MarkdownTable {
  const aligns = cellTexts(lines[startLine + 1]).map(cell => alignOf(cell))
  const header = cellTexts(lines[startLine])
  const rows = lines.slice(startLine + 2, endLine + 1).map(line => cellTexts(line))
  const columns = Math.max(aligns.length, header.length, ...rows.map(row => row.length))
  const fill = <T>(values: T[], blank: T): T[] => [...values, ...Array.from({ length: columns - values.length }, () => blank)]

  return {
    startLine,
    endLine,
    indent: /^\s*/.exec(lines[startLine])?.[0] ?? '',
    aligns: fill(aligns, 'default'),
    header: fill(header, ''),
    rows:   rows.map(row => fill(row, '')),
  }
}

/** Every table of a document, in order, leaving out what is inside fenced code. */
export function findTables (lines: readonly string[]): MarkdownTable[] {
  const tables: MarkdownTable[] = []
  let fence: string | undefined
  for (let index = 0; index < lines.length; index += 1) {
    const marker = /^\s*(```|~~~)/.exec(lines[index])?.[1]
    if (marker) {
      fence = fence === undefined ? marker : (fence === marker ? undefined : fence)
      continue
    }
    if (fence === undefined && lines[index].includes('|') && index + 1 < lines.length && isDelimiterRow(lines[index + 1])) {
      const table = findTable(lines, index + 1)
      if (table) {
        tables.push(table)
        index = table.endLine
      }
    }
  }

  return tables
}
