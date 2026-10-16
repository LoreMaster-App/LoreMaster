/** How a column's text is placed, from the colons of its delimiter row. */
export type ColumnAlign = 'default' | 'left' | 'center' | 'right'

/**
 * A GitHub-flavoured Markdown table: a header row, a delimiter row that gives the alignments, and body rows. Every row has
 * one cell per column, with the cell text as written, trimmed.
 */
export interface MarkdownTable {
  /** The line of the header row, and of the last row. */
  startLine: number
  endLine:   number
  /** The indentation the table is written with. */
  indent:    string
  aligns:    ColumnAlign[]
  header:    string[]
  rows:      string[][]
}

/** A cell of a table: row -1 is the header, and rows count from 0 in the body. */
export interface TableCell {
  row:    number
  column: number
}

/** The number of columns of a table. */
export function columnCount (table: MarkdownTable): number {
  return table.aligns.length
}
