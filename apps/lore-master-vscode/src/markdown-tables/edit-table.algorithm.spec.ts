import type { EditResult, TextPosition } from '../text-edits'
import { editTable, type TableOperation } from './edit-table.algorithm'
import { findTable, findTables, isDelimiterRow, scanCells } from './find-table.algorithm'
import { displayWidth, formatAllTables, formatTable } from './format-table.algorithm'

const at = (line: number, character: number): TextPosition => ({ line, character })

const TABLE = ['| Name | Count |', '|:--|--:|', '| apple | 3 |', '| fig | 12 |']

/** The document after a result's edit, and where the cursor is. */
function run (lines: string[], cursor: TextPosition, operation: TableOperation): { lines: string[], caret: TextPosition } {
  const result: EditResult | undefined = editTable(lines, cursor, operation)
  if (!result) {
    throw new Error(`${operation} did nothing`)
  }
  const [edit] = result.edits

  return {
    lines: [...lines.slice(0, edit.range.start.line), ...edit.text.split('\n'), ...lines.slice(edit.range.end.line + 1)],
    caret: result.selection.active,
  }
}

describe('finding a table', () => {
  it('splits cells at pipes that are not escaped and drops the edge pipes', () => {
    const line = '| a \\| b | `c|d` |'

    expect(scanCells(line).map(span => line.slice(span.start, span.end).trim())).toEqual([String.raw`a \| b`, '`c', 'd`'])
    expect(scanCells('a | b')).toHaveLength(2)
    expect(scanCells('| a |  |')).toHaveLength(2)
  })

  it('knows a delimiter row', () => {
    expect(isDelimiterRow('|:--|--:|')).toBe(true)
    expect(isDelimiterRow('---|:-:')).toBe(true)
    expect(isDelimiterRow('| a | b |')).toBe(false)
    expect(isDelimiterRow('---')).toBe(false)
  })

  it('finds the table from any of its lines, with the alignments, and nothing in plain text', () => {
    const lines = ['text', ...TABLE, 'after']

    expect(findTable(lines, 4)).toMatchObject({ startLine: 1, endLine: 4, aligns: ['left', 'right'], header: ['Name', 'Count'], rows: [['apple', '3'], ['fig', '12']] })
    expect(findTable(lines, 0)).toBeUndefined()
    expect(findTable(['a | b', 'plain | text'], 1)).toBeUndefined()
  })

  it('lists the tables of a document and skips fenced code', () => {
    const lines = [...TABLE, '', '```', '| x |', '|---|', '```', '', '| y |', '|---|', '| 1 |']

    expect(findTables(lines).map(table => [table.startLine, table.endLine])).toEqual([[0, 3], [10, 12]])
  })
})

describe('formatTable', () => {
  it('lines the pipes up and keeps the alignments', () => {
    expect(formatTable(findTable(TABLE, 0)!)).toEqual([
      '| Name  | Count |',
      '| :---- | ----: |',
      '| apple |     3 |',
      '| fig   |    12 |',
    ])
  })

  it('pads short rows, centres and counts wide characters as two columns', () => {
    const lines = ['| a | b |', '|:-:|---|', '| 日本 |', '| x | y | z |']

    expect(formatTable(findTable(lines, 0)!)).toEqual([
      '|  a   | b   |     |',
      '| :--: | --- | --- |',
      '| 日本 |     |     |',
      '|  x   | y   | z   |',
    ])
    expect(displayWidth('日本')).toBe(4)
    expect(displayWidth('é')).toBe(1)
  })

  it('keeps the indentation', () => {
    expect(formatTable(findTable(['  | a |', '  |---|'], 0)!)[0]).toBe('  | a   |')
  })

  it('lists only the tables that need lining up', () => {
    const tidy = formatTable(findTable(TABLE, 0)!)
    const edits = formatAllTables([...TABLE, '', ...tidy])

    expect(edits.map(edit => edit.range.start.line)).toEqual([0])
  })
})

describe('editTable', () => {
  const FORMATTED = ['| Name  | Count |', '| :---- | ----: |', '| apple |     3 |', '| fig   |    12 |']
  const BLANK_ROW = '|       |       |'

  it('lines up the table and keeps the cursor in its cell', () => {
    expect(run(TABLE, at(2, 10), 'format')).toEqual({ lines: FORMATTED, caret: at(2, 14) })
  })

  it('moves to the next cell, then the next row', () => {
    expect(run(TABLE, at(0, 3), 'nextCell').caret).toEqual(at(0, 10))
    expect(run(TABLE, at(0, 12), 'nextCell').caret).toEqual(at(2, 2))
  })

  it('adds a row when Tab leaves the last cell', () => {
    const { lines, caret } = run(TABLE, at(3, 14), 'nextCell')

    expect(lines).toEqual([...FORMATTED, BLANK_ROW])
    expect(caret).toEqual(at(4, 2))
  })

  it('moves back a cell, and up a row', () => {
    expect(run(TABLE, at(2, 10), 'previousCell').caret).toEqual(at(2, 2))
    expect(run(TABLE, at(3, 3), 'previousCell').caret).toEqual(at(2, 14))
    expect(run(TABLE, at(0, 3), 'previousCell').caret).toEqual(at(0, 2))
  })

  it('inserts rows below, above, and below the header', () => {
    expect(run(TABLE, at(2, 3), 'insertRowBelow')).toEqual({ lines: [...FORMATTED.slice(0, 3), BLANK_ROW, FORMATTED[3]], caret: at(3, 2) })
    expect(run(TABLE, at(2, 3), 'insertRowAbove')).toEqual({ lines: [...FORMATTED.slice(0, 2), BLANK_ROW, ...FORMATTED.slice(2)], caret: at(2, 2) })
    expect(run(TABLE, at(0, 3), 'insertRowBelow').lines[2]).toBe(BLANK_ROW)
  })

  it('deletes a row but not the header', () => {
    // The widest row went, so the table closes up around the one left.
    expect(run(TABLE, at(2, 3), 'deleteRow')).toEqual({ lines: ['| Name | Count |', '| :--- | ----: |', '| fig  |    12 |'], caret: at(2, 2) })
    expect(editTable(TABLE, at(0, 3), 'deleteRow')).toBeUndefined()
  })

  it('inserts a column beside the cursor and deletes one, but never the last', () => {
    const inserted = run(TABLE, at(0, 3), 'insertColumnRight')

    expect(inserted.lines[0]).toBe('| Name  |     | Count |')
    expect(inserted.caret).toEqual(at(0, 10))
    expect(run(TABLE, at(0, 3), 'insertColumnLeft').lines[0]).toBe('|     | Name  | Count |')
    expect(run(TABLE, at(2, 3), 'deleteColumn').lines[0]).toBe('| Count |')
    expect(editTable(['| a |', '|---|'], at(0, 3), 'deleteColumn')).toBeUndefined()
  })

  it('does nothing outside a table', () => {
    expect(editTable(['plain'], at(0, 0), 'format')).toBeUndefined()
  })
})
