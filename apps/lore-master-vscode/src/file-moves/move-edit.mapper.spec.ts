import type { FilesPlanMoveResult } from '../engine-protocol'
import { editOperations, summarise } from './move-edit.mapper'

const plan: FilesPlanMoveResult = {
  renames:  [{ from: 'a.b.md', to: 'c.b.md' }],
  edits:    [{ path: 'x.md', line: 2, text: 'parent: c.md' }],
  warnings: [],
}

describe('editOperations', () => {
  it('puts the text edits before the renames', () => {
    expect(editOperations(plan)).toEqual([
      { kind: 'replace-line', path: 'x.md', line: 2, text: 'parent: c.md' },
      { kind: 'rename', from: 'a.b.md', to: 'c.b.md' },
    ])
  })
})

describe('summarise', () => {
  it('says what the plan does', () => {
    expect(summarise(plan)).toBe('rename 1 file that nests under it and update 1 `parent:` line')
    expect(summarise({ ...plan, renames: [{ from: 'a', to: 'b' }, { from: 'c', to: 'd' }], edits: [] })).toBe('rename 2 files that nest under it')
  })
})
