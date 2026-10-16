import { findHeadings } from './find-headings.algorithm'
import { anchorsOf, plainHeadingText } from './heading-anchor.algorithm'
import type { TocOptions } from './toc.contract'

/**
 * The lines of a table of contents for a document: a nested list of links to its headings between the levels asked for.
 * Each level is indented by two spaces (three when numbered) more than the one above, counted from the shallowest level
 * that is actually there, so a document that starts at `##` does not begin with an empty level.
 */
export function renderToc (lines: readonly string[], options: TocOptions): string[] {
  const headings = findHeadings(lines)
  const anchors = anchorsOf(headings)
  const listed = headings
    .map((heading, index) => ({ heading, anchor: anchors[index] }))
    .filter(({ heading }) => heading.level >= options.minLevel && heading.level <= options.maxLevel)
  if (listed.length === 0) {
    return []
  }

  const shallowest = Math.min(...listed.map(({ heading }) => heading.level))
  const counters: number[] = []
  const width = options.ordered ? 3 : 2

  return listed.map(({ heading, anchor }) => {
    const depth = heading.level - shallowest
    counters.length = depth + 1
    for (let level = 0; level <= depth; level += 1) {
      counters[level] ??= 0
    }
    counters[depth] += 1
    const marker = options.ordered ? `${counters[depth]}.` : '-'

    return `${' '.repeat(depth * width)}${marker} [${plainHeadingText(heading.text)}](#${anchor})`
  })
}
