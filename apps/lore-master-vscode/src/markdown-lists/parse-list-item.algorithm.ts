/** One line of a Markdown list, taken apart. */
export interface ListItem {
  /** Everything before the marker: block-quote marks and indentation. */
  prefix:    string
  /** The width of the indentation within the prefix. */
  indent:    number
  /** The marker as written: `-`, `*`, `+`, `3.` or `3)`. */
  marker:    string
  ordered:   boolean
  /** The number of an ordered item. */
  number:    number
  /** `.` or `)` of an ordered item. */
  delimiter: string
  /** The spaces between the marker and the text. */
  spacing:   string
  /** The state of a task item's box: a space or `x`; undefined when there is none. */
  checkbox?: string
  /** The text after the marker (and the box). */
  content:   string
  /** Where the text starts in the line. */
  textStart: number
}

const isSpace = (character: string | undefined): boolean => character === ' ' || character === '\t'

/** The index after the spaces that start at an index. */
function skipSpaces (line: string, from: number): number {
  let index = from
  while (isSpace(line[index])) {
    index += 1
  }

  return index
}

/** The marker that starts at an index, and where it ends: `-`, `*`, `+`, or up to nine digits and a `.` or `)`. */
function markerAt (line: string, from: number): { marker: string, digits: string, delimiter: string } | undefined {
  if ('-*+'.includes(line[from] ?? ' ')) {
    return { marker: line[from], digits: '', delimiter: '' }
  }
  let end = from
  while (end < line.length && end - from < 9 && line[end] >= '0' && line[end] <= '9') {
    end += 1
  }
  const delimiter = line[end]
  if (end === from || (delimiter !== '.' && delimiter !== ')')) {
    return undefined
  }

  return { marker: line.slice(from, end + 1), digits: line.slice(from, end), delimiter }
}

/** Takes a line apart as a list item, or says it is not one. */
export function parseListItem (line: string): ListItem | undefined {
  // Block-quote marks, each with the spaces before it, come first.
  let quoteEnd = 0
  for (let index = skipSpaces(line, 0); line[index] === '>'; index = skipSpaces(line, quoteEnd)) {
    // One space after the mark belongs to the quote, not to the list inside it.
    quoteEnd = index + (line[index + 1] === ' ' ? 2 : 1)
  }
  const markerStart = skipSpaces(line, quoteEnd)
  const found = markerAt(line, markerStart)
  const afterMarker = markerStart + (found?.marker.length ?? 0)
  if (!found || !isSpace(line[afterMarker])) {
    return undefined
  }

  const textStart = skipSpaces(line, afterMarker)
  const spacing = line.slice(afterMarker, textStart)
  const rest = line.slice(textStart)
  // A rule made of the same mark (`* * *`, `- - -`) is not an item.
  if (!found.digits && rest.length > 0 && [...rest].every(character => character === found.marker || isSpace(character)) && rest.includes(found.marker)) {
    return undefined
  }

  const box = /^\[([ x])\](?: |\t|$)/i.exec(rest)
  const checkbox = box?.[1]
  const content = box ? rest.slice(box[0].length).trimStart() : rest
  const boxWidth = box ? rest.length - content.length : 0

  return {
    prefix:    line.slice(0, markerStart),
    indent:    markerStart - quoteEnd,
    marker:    found.marker,
    ordered:   found.digits !== '',
    number:    found.digits === '' ? 0 : Number(found.digits),
    delimiter: found.delimiter,
    spacing,
    checkbox,
    content,
    textStart: textStart + boxWidth,
  }
}

/** Where a line's marker ends, which is as far left as an Enter in the middle of the line can split it. */
export function markerEnd (item: ListItem): number {
  return item.prefix.length + item.marker.length + item.spacing.length
}
