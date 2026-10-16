import { followLines, linesOf, replaceLines, type EditResult, type TextSelection } from '../text-edits'
import { parseListItem } from './parse-list-item.algorithm'
import { renumberOrdered } from './renumber-list.algorithm'

/** The list item above a line, in the same block, which an item can be nested under. */
function itemAbove (lines: readonly string[], index: number): ReturnType<typeof parseListItem> {
  for (let above = index - 1; above >= 0; above -= 1) {
    const item = parseListItem(lines[above])
    if (item) {
      return item
    }
    if (!/^\s{2,}\S/.test(lines[above])) {
      return undefined
    }
  }

  return undefined
}

/** The item before a line that sits at an indentation, skipping what is nested deeper, in the same block. */
function siblingAbove (lines: readonly string[], index: number, indent: number): ReturnType<typeof parseListItem> {
  for (let above = index - 1; above >= 0; above -= 1) {
    const item = parseListItem(lines[above])
    if (!item && !/^\s{2,}\S/.test(lines[above])) {
      return undefined
    }
    if (item && item.indent <= indent) {
      return item.indent === indent ? item : undefined
    }
  }

  return undefined
}

/**
 * Nests the list items among the selected lines one level deeper, or brings them one level out, by the width of their own
 * marker (two for `-`, three for `1.`), and numbers the lists again: an item that starts a nested list starts at 1, and the
 * items that follow close up the gap. An item with nothing above it cannot be nested, and one at the left edge cannot come out.
 */
export function indentListItems (lines: readonly string[], selection: TextSelection, direction: 'indent' | 'outdent'): EditResult | undefined {
  const { from, to } = linesOf(selection)
  const next = [...lines]

  for (let index = from; index <= to; index += 1) {
    const item = parseListItem(next[index])
    if (!item) {
      continue
    }
    const unit = item.marker.length + item.spacing.length
    const quoteLength = item.prefix.length - item.indent
    if (direction === 'outdent') {
      const remove = Math.min(item.indent, unit)
      next[index] = remove === 0 ? next[index] : next[index].slice(0, quoteLength) + next[index].slice(quoteLength + remove)
      // An item that comes out follows the one before it at its new level.
      const sibling = remove === 0 ? undefined : siblingAbove(next, index, item.indent - remove)
      if (item.ordered && sibling?.ordered) {
        next[index] = next[index].replace(/^((?:\s*>)*\s*)\d+/, (_match, before: string) => `${before}${sibling.number + 1}`)
      }
      continue
    }

    const above = itemAbove(next, index)
    if (!above) {
      continue
    }
    const indented = next[index].slice(0, quoteLength) + ' '.repeat(unit) + next[index].slice(quoteLength)
    // The first item of a nested list starts it at 1, whatever number it had.
    const startsNested = item.ordered && above.indent < item.indent + unit
    next[index] = startsNested ? indented.replace(/^((?:\s*>)*\s*)\d+/, (_match, before: string) => `${before}1`) : indented
  }

  if (next.every((line, index) => line === lines[index])) {
    return undefined
  }
  for (const [index, text] of renumberOrdered(next, from)) {
    next[index] = text
  }
  let last = to
  for (let index = to + 1; index < next.length; index += 1) {
    if (next[index] !== lines[index]) {
      last = index
    }
  }

  return replaceLines(lines, from, last, next.slice(from, last + 1), followLines(selection, lines, next))
}
