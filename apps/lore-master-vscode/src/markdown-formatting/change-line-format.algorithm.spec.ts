import type { EditResult, TextSelection } from '../text-edits'
import { decreaseHeading, increaseHeading, toggleQuote, toggleTask } from './change-line-format.algorithm'
import { toggleCodeBlock } from './toggle-code-block.algorithm'

const cursor = (line: number, character = 0): TextSelection => ({ anchor: { line, character }, active: { line, character } })
const range = (from: number, to: number, toCharacter = 0): TextSelection => ({ anchor: { line: from, character: 0 }, active: { line: to, character: toCharacter } })

/** The lines after the result's edit. */
function applied (lines: string[], result: EditResult | undefined): string[] {
  if (!result) {
    throw new Error('nothing was done')
  }
  const [edit] = result.edits

  return [...lines.slice(0, edit.range.start.line), ...edit.text.split('\n'), ...lines.slice(edit.range.end.line + 1)]
}

const quote = (lines: string[], selection: TextSelection): string[] => applied(lines, toggleQuote(lines, selection))
const task = (lines: string[], selection = cursor(0)): string[] => applied(lines, toggleTask(lines, selection))
const deeper = (line: string): string[] => applied([line], increaseHeading([line], cursor(0)))
const shallower = (line: string): string[] => applied([line], decreaseHeading([line], cursor(0)))
const fence = (lines: string[], selection: TextSelection): string[] => applied(lines, toggleCodeBlock(lines, selection))

describe('toggleQuote', () => {
  it('quotes the selected lines and takes the quote off again', () => {
    const quoted = quote(['one', '', 'two'], range(0, 2, 3))

    expect(quoted).toEqual(['> one', '>', '> two'])
    expect(quote(quoted, range(0, 2, 3))).toEqual(['one', '', 'two'])
  })

  it('does not count a last line the selection only just enters', () => {
    expect(quote(['one', 'two'], range(0, 1, 0))).toEqual(['> one', 'two'])
  })

  it('keeps the cursor on the same text', () => {
    expect(toggleQuote(['hello'], cursor(0, 3))?.selection.active).toEqual({ line: 0, character: 5 })
  })
})

describe('toggleTask', () => {
  it('cycles a plain line through unchecked, checked and unchecked', () => {
    const first = task(['buy milk'])
    const second = task(first)

    expect(first).toEqual(['- [ ] buy milk'])
    expect(second).toEqual(['- [x] buy milk'])
    expect(task(second)).toEqual(['- [ ] buy milk'])
  })

  it('gives a list item a box, keeping its marker and indentation', () => {
    expect(task(['  - one'])).toEqual(['  - [ ] one'])
    expect(task(['1. one'])).toEqual(['1. [ ] one'])
  })

  it('ticks every line of a selection together and leaves blank lines', () => {
    expect(task(['- [ ] a', '', '- [ ] b'], range(0, 2, 7))).toEqual(['- [x] a', '', '- [x] b'])
  })

  it('makes a mixed selection all unchecked tasks', () => {
    expect(task(['- [x] a', 'b'], range(0, 1, 1))).toEqual(['- [ ] a', '- [ ] b'])
  })
})

describe('headings', () => {
  it('makes a plain line a heading and goes deeper up to six', () => {
    expect(deeper('Title')).toEqual(['# Title'])
    expect(deeper('## Title')).toEqual(['### Title'])
    expect(increaseHeading(['###### Title'], cursor(0))).toBeUndefined()
  })

  it('goes shallower and turns a first-level heading back into text', () => {
    expect(shallower('### Title')).toEqual(['## Title'])
    expect(shallower('# Title')).toEqual(['Title'])
    expect(decreaseHeading(['Title'], cursor(0))).toBeUndefined()
  })
})

describe('toggleCodeBlock', () => {
  it('fences the selected lines and selects what was inside', () => {
    const lines = ['a', 'b', 'c']
    const result = toggleCodeBlock(lines, range(1, 1, 1))

    expect(applied(lines, result)).toEqual(['a', '```', 'b', '```', 'c'])
    expect(result?.selection).toEqual({ anchor: { line: 2, character: 0 }, active: { line: 2, character: 1 } })
  })

  it('opens an empty block on an empty line and puts the cursor in it', () => {
    const result = toggleCodeBlock([''], cursor(0))

    expect(applied([''], result)).toEqual(['```', '', '```'])
    expect(result?.selection.active).toEqual({ line: 1, character: 0 })
  })

  it('takes the fences off when they are selected, or when the cursor is inside the block', () => {
    const fenced = ['```ts', 'x', '```']

    expect(fence(fenced, range(0, 2, 3))).toEqual(['x'])
    expect(fence(fenced, cursor(1))).toEqual(['x'])
  })
})
