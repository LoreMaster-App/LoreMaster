import { groupNotes, isOverdue } from './group-notes.algorithm'
import type { FoundNote } from './notes.contract'

const note = (path: string, line: number, tag: string, extra: Partial<FoundNote> = {}): FoundNote =>
  ({ folder: '/w', path, line, column: 4, tag, text: `${tag} in ${path}`, ...extra })

const notes = [note('b.ts', 9, 'FIXME'), note('a.ts', 5, 'TODO'), note('a.ts', 2, 'TODO'), note('c.go', 1, 'WHY'), note('a.ts', 7, 'BUG')]

describe('groupNotes', () => {
  it('groups by tag in the order of the tag list, leaves a tag with no note out and puts an unlisted tag last', () => {
    const groups = groupNotes(notes, ['TODO', 'NOTE', 'FIXME', 'BUG'], 'tag')

    expect(groups.map(group => `${group.label}:${group.notes.length}`)).toEqual(['TODO:2', 'FIXME:1', 'BUG:1', 'WHY:1'])
    expect(groups[0].notes.map(each => `${each.path}:${each.line}`)).toEqual(['a.ts:2', 'a.ts:5'])
    expect(groups[0].tag).toBe('TODO')
  })

  it('groups by file in path order, with the notes of a file in line order', () => {
    const groups = groupNotes(notes, [], 'file')

    expect(groups.map(group => group.label)).toEqual(['a.ts', 'b.ts', 'c.go'])
    expect(groups[0].notes.map(each => each.line)).toEqual([2, 5, 7])
    expect(groups[0].tag).toBeUndefined()
  })

  it('names a file after its folder when there is more than one folder', () => {
    const many = [note('a.ts', 1, 'TODO', { folder: '/repos/two' }), note('a.ts', 1, 'TODO', { folder: '/repos/one/' })]

    expect(groupNotes(many, [], 'file').map(group => group.label)).toEqual(['one/a.ts', 'two/a.ts'])
  })

  it('has no groups when there are no notes', () => {
    expect(groupNotes([], ['TODO'], 'tag')).toEqual([])
  })
})

describe('isOverdue', () => {
  it('is true only when the date has passed', () => {
    expect(isOverdue(note('a.ts', 1, 'TODO', { due: '2026-01-01' }), '2026-10-23')).toBe(true)
    expect(isOverdue(note('a.ts', 1, 'TODO', { due: '2026-10-23' }), '2026-10-23')).toBe(false)
    expect(isOverdue(note('a.ts', 1, 'TODO'), '2026-10-23')).toBe(false)
  })
})
