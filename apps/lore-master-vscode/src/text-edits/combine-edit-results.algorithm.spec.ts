import { combineEditResults } from './combine-edit-results.algorithm'
import type { EditResult } from './text-edit.contract'

const at = (line: number, character: number): { line: number, character: number } => ({ line, character })
const result = (from: [number, number], to: [number, number], text: string, cursor: [number, number]): EditResult => ({
  edits:     [{ range: { start: at(...from), end: at(...to) }, text }],
  selection: { anchor: at(...cursor), active: at(...cursor) },
})

describe('combineEditResults', () => {
  it('keeps every edit and moves a cursor along the same line past an earlier edit', () => {
    // "a b" with ** put around both: the second cursor ends up after the four characters the first edit added.
    const first = result([0, 0], [0, 1], '**a**', [0, 3])
    const second = result([0, 2], [0, 3], '**b**', [0, 5])

    const combined = combineEditResults([second, first])

    expect(combined.edits).toHaveLength(2)
    expect(combined.selections).toEqual([
      { anchor: at(0, 3), active: at(0, 3) },
      { anchor: at(0, 9), active: at(0, 9) },
    ])
  })

  it('moves a cursor down by the lines an earlier edit added, and up by those it removed', () => {
    const added = result([0, 0], [0, 1], 'x\ny\nz', [2, 1])
    const below = result([3, 0], [3, 1], 'w', [3, 1])
    expect(combineEditResults([added, below]).selections[1].active).toEqual(at(5, 1))

    const removed = result([0, 0], [2, 1], 'x', [0, 1])
    expect(combineEditResults([removed, result([3, 0], [3, 1], 'w', [3, 1])]).selections[1].active).toEqual(at(1, 1))
  })

  it('drops a result that would overlap an earlier one', () => {
    const combined = combineEditResults([result([0, 0], [0, 5], 'a', [0, 1]), result([0, 3], [0, 8], 'b', [0, 4])])

    expect(combined.edits).toHaveLength(1)
  })
})
