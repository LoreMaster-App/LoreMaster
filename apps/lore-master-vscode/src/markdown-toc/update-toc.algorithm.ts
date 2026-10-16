import type { TextEdit, TextPosition } from '../text-edits'
import { renderToc } from './render-toc.algorithm'
import { TOC_END, TOC_START, type TocOptions } from './toc.contract'

/** The lines of the first table of contents block of a document: the start mark and the end mark. */
function findBlock (lines: readonly string[]): { start: number, end: number } | undefined {
  const start = lines.findIndex(line => line.trim() === TOC_START)
  if (start === -1) {
    return undefined
  }
  const end = lines.findIndex((line, index) => index > start && line.trim() === TOC_END)

  return end === -1 ? undefined : { start, end }
}

/** Whether a document has a table of contents block, for the save that keeps it current. */
export function hasToc (lines: readonly string[]): boolean {
  return findBlock(lines) !== undefined
}

/** The edit that makes a document's table of contents list its headings as they are now; nothing when it already does. */
export function updateToc (lines: readonly string[], options: TocOptions): TextEdit | undefined {
  const block = findBlock(lines)
  if (!block) {
    return undefined
  }
  // The table of contents is not part of what it lists, and it holds no headings of its own.
  const outside = [...lines.slice(0, block.start), ...lines.slice(block.end + 1)]
  const list = renderToc(outside, options)
  const current = lines.slice(block.start + 1, block.end)
  if (list.length === current.length && list.every((line, index) => line === current[index])) {
    return undefined
  }

  return {
    range: { start: { line: block.start + 1, character: 0 }, end: { line: block.end, character: 0 } },
    text:  list.length === 0 ? '' : `${list.join('\n')}\n`,
  }
}

/**
 * The edit that writes a table of contents at a line: the marks around the list. When the document already has one, it is
 * brought up to date in place instead and nothing new is added.
 */
export function insertToc (lines: readonly string[], at: TextPosition, options: TocOptions): TextEdit | undefined {
  if (hasToc(lines)) {
    return updateToc(lines, options)
  }
  const list = renderToc(lines, options)
  const body = list.length === 0 ? '' : `${list.join('\n')}\n`
  const here = lines[at.line] ?? ''
  const lead = here.trim() === '' ? '' : '\n\n'
  const where: TextPosition = here.trim() === '' ? { line: at.line, character: 0 } : { line: at.line, character: here.length }

  return { range: { start: where, end: where }, text: `${lead}${TOC_START}\n${body}${TOC_END}\n` }
}
