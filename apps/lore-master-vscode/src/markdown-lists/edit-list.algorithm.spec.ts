import type { EditResult, TextPosition, TextSelection } from '../text-edits'
import { continueList } from './continue-list.algorithm'
import { indentListItems } from './indent-list.algorithm'
import { parseListItem } from './parse-list-item.algorithm'
import { renumberOrdered } from './renumber-list.algorithm'

const at = (line: number, character: number): TextPosition => ({ line, character })
const cursor = (line: number, character: number): TextSelection => ({ anchor: at(line, character), active: at(line, character) })

function offsetOf (lines: string[], position: TextPosition): number {
  return lines.slice(0, position.line).reduce((sum, line) => sum + line.length + 1, 0) + position.character
}

/** The document after every edit of a result, which all refer to the document as it was. */
function applied (lines: string[], result: EditResult | undefined): string[] {
  if (!result) {
    throw new Error('nothing was done')
  }
  const edits = [...result.edits].sort((left, right) => offsetOf(lines, right.range.start) - offsetOf(lines, left.range.start))
  let next = lines.join('\n')
  for (const edit of edits) {
    next = next.slice(0, offsetOf(lines, edit.range.start)) + edit.text + next.slice(offsetOf(lines, edit.range.end))
  }

  return next.split('\n')
}

/** Enter at a place in some lines. */
const enter = (lines: string[], line: number, character: number): string[] => applied(lines, continueList(lines, at(line, character)))

/** Tab or Shift+Tab with a selection. */
const nest = (lines: string[], selection: TextSelection, direction: 'indent' | 'outdent'): string[] => applied(lines, indentListItems(lines, selection, direction))

describe('parseListItem', () => {
  it('takes bullets, numbers and tasks apart', () => {
    expect(parseListItem('- one')).toMatchObject({ marker: '-', ordered: false, content: 'one', indent: 0 })
    expect(parseListItem('   12) two')).toMatchObject({ marker: '12)', ordered: true, number: 12, delimiter: ')', indent: 3 })
    expect(parseListItem('* [x] done')).toMatchObject({ checkbox: 'x', content: 'done' })
    expect(parseListItem('> - quoted')).toMatchObject({ prefix: '> ', indent: 0, content: 'quoted' })
  })

  it('takes a task with no text and an item with no text', () => {
    expect(parseListItem('- [ ] ')).toMatchObject({ checkbox: ' ', content: '' })
    expect(parseListItem('- ')).toMatchObject({ content: '' })
  })

  it('does not take rules, emphasis or plain text for items', () => {
    for (const line of ['---', '* * *', '- - -', '**bold**', '-no space', '1.5 units', 'plain', '']) {
      expect(parseListItem(line)).toBeUndefined()
    }
  })

  it('is not slowed down by a very long run of spaces', () => {
    const started = Date.now()
    parseListItem(`-${' '.repeat(100_000)}x`)
    parseListItem(' '.repeat(100_000))

    expect(Date.now() - started).toBeLessThan(500)
  })
})

describe('continueList (Enter)', () => {
  it('starts the next item, numbered and with an empty box for a task', () => {
    expect(enter(['- one'], 0, 5)).toEqual(['- one', '- '])
    expect(enter(['3. three'], 0, 8)).toEqual(['3. three', '4. '])
    expect(enter(['- [x] done'], 0, 10)).toEqual(['- [x] done', '- [ ] '])
    expect(enter(['  * nested'], 0, 10)).toEqual(['  * nested', '  * '])
  })

  it('carries what is after the cursor into the new item and puts the cursor at its start', () => {
    const result = continueList(['- one two'], at(0, 5))

    expect(applied(['- one two'], result)).toEqual(['- one', '- two'])
    expect(result?.selection.active).toEqual(at(1, 2))
  })

  it('numbers the items below again', () => {
    expect(enter(['1. a', '2. b', '3. c'], 0, 4)).toEqual(['1. a', '2. ', '3. b', '4. c'])
  })

  it('ends the list on an empty item, and brings a nested one out a level', () => {
    expect(enter(['- one', '- '], 1, 2)).toEqual(['- one', ''])
    expect(enter(['- one', '  - '], 1, 4)).toEqual(['- one', '- '])
  })

  it('leaves the key alone off an item or inside the marker', () => {
    expect(continueList(['plain'], at(0, 5))).toBeUndefined()
    expect(continueList(['- one'], at(0, 1))).toBeUndefined()
  })

  it('keeps a block quote around the new item', () => {
    expect(enter(['> - one'], 0, 7)).toEqual(['> - one', '> - '])
  })
})

describe('indentListItems (Tab and Shift+Tab)', () => {
  it('nests an item by the width of its marker and starts a nested numbered list at 1', () => {
    expect(nest(['- a', '- b'], cursor(1, 3), 'indent')).toEqual(['- a', '  - b'])
    expect(nest(['1. a', '2. b', '3. c'], cursor(1, 4), 'indent')).toEqual(['1. a', '   1. b', '2. c'])
  })

  it('cannot nest the first item', () => {
    expect(indentListItems(['- a', '- b'], cursor(0, 3), 'indent')).toBeUndefined()
  })

  it('brings an item out and numbers it as the next one of the list it joins', () => {
    expect(nest(['1. a', '   1. b', '   2. c'], cursor(1, 7), 'outdent')).toEqual(['1. a', '2. b', '   2. c'])
    expect(indentListItems(['- a'], cursor(0, 3), 'outdent')).toBeUndefined()
  })

  it('works on every item of a selection', () => {
    expect(nest(['- a', '- b', '- c'], { anchor: at(1, 0), active: at(2, 3) }, 'indent')).toEqual(['- a', '  - b', '  - c'])
  })

  it('moves the cursor with the text', () => {
    expect(indentListItems(['- a', '- b'], cursor(1, 3), 'indent')?.selection.active).toEqual(at(1, 5))
  })
})

describe('renumberOrdered', () => {
  it('numbers each level in order, keeping the number a list starts at', () => {
    const lines = ['3. a', '9. b', '   1. x', '   5. y', '9. c']

    expect([...renumberOrdered(lines, 0)]).toEqual([[1, '4. b'], [3, '   2. y'], [4, '5. c']])
  })

  it('keeps a list written with every number the same', () => {
    expect(renumberOrdered(['1. a', '1. b', '1. c'], 0).size).toBe(0)
  })

  it('stops at a blank line and changes only lines from where it is told', () => {
    expect([...renumberOrdered(['1. a', '5. b', '', '7. c'], 0)]).toEqual([[1, '2. b']])
    expect(renumberOrdered(['1. a', '5. b'], 2).size).toBe(0)
  })
})
