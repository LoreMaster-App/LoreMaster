import { parseListItem } from './parse-list-item.algorithm'

/** The numbering of one list at one level while a block is walked. */
interface Level {
  indent:  number
  ordered: boolean
  /** The next number to give. */
  next:    number
  /** Every number the same (`1.` `1.` `1.`), which is a way of writing a list that is kept. */
  allSame: boolean
  /** The number of the list's first item. */
  start:   number
  /** How many items the level has seen. */
  count:   number
}

/** Whether a line belongs to the list block that runs through it: an item, or indented text under one. */
function inBlock (line: string): boolean {
  return parseListItem(line) !== undefined || /^\s{2,}\S/.test(line)
}

/**
 * The lines of a list block with its ordered items numbered in order, level by level. A list keeps the number it
 * starts at, and one that is written with every number the same (`1.` throughout) stays that way. Only lines at or after
 * `fromLine` are changed; the result says which lines that is, as new text by line number.
 */
export function renumberOrdered (lines: readonly string[], fromLine: number): Map<number, string> {
  let start = fromLine
  while (start > 0 && inBlock(lines[start - 1])) {
    start -= 1
  }
  const changes = new Map<number, string>()
  const levels: Level[] = []

  for (let index = start; index < lines.length && inBlock(lines[index]); index += 1) {
    const item = parseListItem(lines[index])
    if (!item) {
      continue
    }
    let level = levels.at(-1)
    while (level && level.indent > item.indent) {
      levels.pop()
      level = levels.at(-1)
    }
    if (!level || level.indent < item.indent || level.ordered !== item.ordered) {
      if (level && level.indent === item.indent) {
        levels.pop()
      }
      level = { indent: item.indent, ordered: item.ordered, next: item.number, allSame: false, start: item.number, count: 0 }
      levels.push(level)
    }
    if (!item.ordered) {
      continue
    }

    let number = level.next
    if (level.count === 1) {
      level.allSame = item.number === level.start
    }
    if (level.allSame) {
      number = level.start
    }
    level.count += 1
    level.next = level.allSame ? level.start : number + 1

    if (index >= fromLine && number !== item.number) {
      changes.set(index, `${item.prefix}${number}${item.delimiter}${item.spacing}${lines[index].slice(item.prefix.length + item.marker.length + item.spacing.length)}`)
    }
  }

  return changes
}
