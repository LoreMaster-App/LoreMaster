import { followLines, linesOf, replaceLines, type EditResult, type TextSelection } from '../text-edits'

const QUOTE = /^(\s*)> ?/
const TASK = /^(\s*(?:[-*+]|\d+[.)])\s+)\[([ x])\]\s/i
const LIST_ITEM = /^(\s*(?:[-*+]|\d+[.)])\s+)/
const HEADING = /^(#{1,6}) +(\S.*)$/

/** What a command that works a line at a time does to each line it is given. */
type LineChange = (lines: readonly string[]) => string[]

function changeLines (lines: readonly string[], selection: TextSelection, change: LineChange): EditResult | undefined {
  const { from, to } = linesOf(selection)
  const before = lines.slice(from, to + 1)
  const after = change(before)
  if (after.every((line, index) => line === before[index])) {
    return undefined
  }
  const next = [...lines]
  next.splice(from, before.length, ...after)

  return replaceLines(lines, from, to, after, followLines(selection, lines, next))
}

/** Puts the lines in a block quote, or takes them out of one when every one of them is already quoted. */
export function toggleQuote (lines: readonly string[], selection: TextSelection): EditResult | undefined {
  return changeLines(lines, selection, block => {
    const filled = block.filter(line => line.trim() !== '')
    const allQuoted = filled.length > 0 && filled.every(line => QUOTE.test(line))

    return block.map(line => {
      if (allQuoted) {
        return line.replace(QUOTE, '$1')
      }

      return line.trim() === '' ? '>' : `> ${line}`
    })
  })
}

/**
 * Cycles the lines through a to-do list: when every line is an unchecked task they are all checked, when every
 * one is checked they are all unchecked, and otherwise every line becomes an unchecked task (a list item gets
 * a box, a plain line becomes an item with one). Pressing it repeatedly goes round.
 */
export function toggleTask (lines: readonly string[], selection: TextSelection): EditResult | undefined {
  return changeLines(lines, selection, block => {
    const filled = block.filter(line => line.trim() !== '')
    const allUnchecked = filled.length > 0 && filled.every(line => TASK.exec(line)?.[2] === ' ')
    const allChecked = filled.length > 0 && filled.every(line => /x/i.test(TASK.exec(line)?.[2] ?? ''))

    return block.map(line => {
      if (line.trim() === '') {
        return line
      }
      if (allUnchecked) {
        return line.replace(TASK, '$1[x] ')
      }
      if (allChecked) {
        return line.replace(TASK, '$1[ ] ')
      }
      if (TASK.test(line)) {
        return line.replace(TASK, '$1[ ] ')
      }
      const item = LIST_ITEM.exec(line)
      if (item) {
        return `${item[1]}[ ] ${line.slice(item[1].length)}`
      }
      const indent = /^\s*/.exec(line)?.[0] ?? ''

      return `${indent}- [ ] ${line.slice(indent.length)}`
    })
  })
}

/** Makes the headings among the lines one level deeper (`#` to `##`), or turns a plain line into a first-level heading. */
export function increaseHeading (lines: readonly string[], selection: TextSelection): EditResult | undefined {
  return changeLines(lines, selection, block => block.map(line => {
    if (line.trim() === '') {
      return line
    }
    const heading = HEADING.exec(line)
    if (!heading) {
      return `# ${line}`
    }

    return heading[1].length < 6 ? `#${line}` : line
  }))
}

/** Makes the headings among the lines one level shallower, and turns a first-level heading back into text. */
export function decreaseHeading (lines: readonly string[], selection: TextSelection): EditResult | undefined {
  return changeLines(lines, selection, block => block.map(line => {
    const heading = HEADING.exec(line)
    if (!heading) {
      return line
    }

    return heading[1].length > 1 ? line.slice(1) : heading[2]
  }))
}
