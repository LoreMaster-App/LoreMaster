import type { TextEdit } from '../text-edits'
import { findHeadings } from './find-headings.algorithm'
import type { TocOptions } from './toc.contract'

/** A number a heading was given: `1`, `1.2`, `2.1.3.`, with the spaces after it. */
const SECTION_NUMBER = /^\d+(?:\.\d+)*\.?[ \t]+/

/** The `#` marks a heading line starts with. */
const MARKS = /^ {0,3}#{1,6}/

/**
 * The edits that give the headings between the levels of the options their section numbers (`1`, `1.1`, `1.2`, `2`...),
 * replacing any numbers they have, so running it again after the headings change is the same as running it once. Only
 * headings written with `#` marks are numbered.
 */
export function addSectionNumbers (lines: readonly string[], options: TocOptions): TextEdit[] {
  const counters: number[] = []
  const edits: TextEdit[] = []
  for (const heading of findHeadings(lines)) {
    if (!heading.atx || heading.level < options.minLevel || heading.level > options.maxLevel) {
      continue
    }
    const depth = heading.level - options.minLevel
    counters.length = Math.min(counters.length, depth + 1)
    // A level that was skipped (a `####` straight under a `##`) counts as the first of its level.
    while (counters.length < depth) {
      counters.push(1)
    }
    counters[depth] = (counters[depth] ?? 0) + 1
    edits.push(...renumber(lines, heading.line, counters, depth))
  }

  return edits
}

function renumber (lines: readonly string[], line: number, counters: number[], depth: number): TextEdit[] {
  const marks = MARKS.exec(lines[line])?.[0]
  if (marks === undefined) {
    return []
  }
  const title = lines[line].slice(marks.length).trimStart().replace(SECTION_NUMBER, '')
  const next = `${marks} ${counters.slice(0, depth + 1).join('.')} ${title}`

  return next === lines[line] ? [] : [{ range: { start: { line, character: 0 }, end: { line, character: lines[line].length } }, text: next }]
}

/** The edits that take section numbers off the headings between the levels of the options. */
export function removeSectionNumbers (lines: readonly string[], options: TocOptions): TextEdit[] {
  const edits: TextEdit[] = []
  for (const heading of findHeadings(lines)) {
    if (!heading.atx || heading.level < options.minLevel || heading.level > options.maxLevel) {
      continue
    }
    const marks = MARKS.exec(lines[heading.line])?.[0]
    const text = marks === undefined ? '' : lines[heading.line].slice(marks.length).trimStart()
    const title = text.replace(SECTION_NUMBER, '')
    if (marks !== undefined && title !== text) {
      edits.push({ range: { start: { line: heading.line, character: 0 }, end: { line: heading.line, character: lines[heading.line].length } }, text: `${marks} ${title}` })
    }
  }

  return edits
}
