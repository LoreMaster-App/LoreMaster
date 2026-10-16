/** One heading of a document. */
export interface DocumentHeading {
  /** The line it is on (for a setext heading, the line of its text). */
  line:  number
  level: number
  /** The heading as written, without its `#` marks. */
  text:  string
  /** Written with `#` marks, as opposed to underlined with `===` or `---`. */
  atx:   boolean
}

const ATX = /^ {0,3}(#{1,6})([ \t].*)?$/
const CLOSING_MARKS = /[ \t]+#+$/
const SETEXT_UNDERLINE = /^ {0,3}(=+|-+)[ \t]*$/
/** A `#` heading's level and text, with the optional closing `#` marks left off; nothing when the line is not one or has no text. */
function parseAtx (line: string): { level: number, text: string } | undefined {
  const match = ATX.exec(line)
  const text = (match?.[2] ?? '').trim().replace(CLOSING_MARKS, '').trim()

  return match && text !== '' ? { level: match[1].length, text } : undefined
}

const FENCE = /^ {0,3}(?:```|~~~)/

/** The line a document's front matter ends on, or -1 when there is none. */
function frontMatterEnd (lines: readonly string[]): number {
  if (lines[0]?.trim() !== '---') {
    return -1
  }
  const end = lines.findIndex((line, index) => index > 0 && (line.trim() === '---' || line.trim() === '...'))

  return end
}

/**
 * The headings of a document in order, leaving out front matter and anything inside fenced code. Headings are written
 * with `#` marks or underlined with `===` (level 1) or `---` (level 2).
 */
export function findHeadings (lines: readonly string[]): DocumentHeading[] {
  const headings: DocumentHeading[] = []
  let fenced = false
  for (let index = frontMatterEnd(lines) + 1; index < lines.length; index += 1) {
    const line = lines[index]
    if (FENCE.test(line)) {
      fenced = !fenced
      continue
    }
    if (fenced) {
      continue
    }
    const atx = parseAtx(line)
    if (atx) {
      headings.push({ line: index, level: atx.level, text: atx.text, atx: true })
      continue
    }
    const underline = SETEXT_UNDERLINE.exec(lines[index + 1] ?? '')
    // A paragraph starts at the top, after a blank line, or straight after a code block.
    const startsParagraph = index === 0 || lines[index - 1].trim() === '' || FENCE.test(lines[index - 1])
    if (underline && startsParagraph && line.trim() !== '' && !/^\s*(?:[-*+]|\d+[.)])\s/.test(line)) {
      headings.push({ line: index, level: underline[1].startsWith('=') ? 1 : 2, text: line.trim(), atx: false })
      index += 1
    }
  }

  return headings
}
