import type { TextEdit } from '../text-edits'
import { findHeadings } from './find-headings.algorithm'
import { anchorsOf, githubSlug, plainHeadingText } from './heading-anchor.algorithm'
import { addSectionNumbers, removeSectionNumbers } from './number-sections.algorithm'
import { renderToc } from './render-toc.algorithm'
import type { TocOptions } from './toc.contract'
import { insertToc, updateToc } from './update-toc.algorithm'

const options: TocOptions = { minLevel: 2, maxLevel: 4, ordered: false }

/** The lines after the edits, which all refer to the lines as they were. */
function applied (lines: string[], edits: (TextEdit | undefined)[]): string[] {
  const offsetOf = (position: { line: number, character: number }): number => lines.slice(0, position.line).reduce((sum, line) => sum + line.length + 1, 0) + position.character
  let text = lines.join('\n')
  const real = edits.filter((each): each is TextEdit => each !== undefined)
  const backToFront = real.sort((left, right) => offsetOf(right.range.start) - offsetOf(left.range.start))
  for (const edit of backToFront) {
    text = text.slice(0, offsetOf(edit.range.start)) + edit.text + text.slice(offsetOf(edit.range.end))
  }

  return text.split('\n')
}

describe('findHeadings', () => {
  it('finds # headings and underlined ones, and skips code and front matter', () => {
    const lines = ['---', '# not a heading', '---', '# Title', '', 'Setext', '======', '', '## Two ##', '```', '# in code', '```', 'Sub', '---', '- item', '---']

    expect(findHeadings(lines).map(heading => [heading.level, heading.text, heading.atx])).toEqual([
      [1, 'Title', true], [1, 'Setext', false], [2, 'Two', true], [2, 'Sub', false],
    ])
  })

  it('does not take a list item above a rule for a heading', () => {
    expect(findHeadings(['- item', '---'])).toEqual([])
  })
})

describe('anchors', () => {
  it('follows GitHub: lower case, punctuation gone, spaces to hyphens', () => {
    expect(githubSlug('Getting Started!')).toBe('getting-started')
    expect(githubSlug('The `engine` & the **editor**')).toBe('the-engine--the-editor')
    expect(githubSlug('snake_case and _italic_')).toBe('snake_case-and-italic')
    expect(githubSlug('Café déjà vu')).toBe('café-déjà-vu')
    expect(githubSlug('See [the guide](guide.md)')).toBe('see-the-guide')
  })

  it('numbers a repeated heading as GitHub does', () => {
    const headings = findHeadings(['## Setup', '## Setup', '## Setup'])

    expect(anchorsOf(headings)).toEqual(['setup', 'setup-1', 'setup-2'])
    expect(plainHeadingText('a **b** [c](d)')).toBe('a b c')
  })
})

describe('renderToc', () => {
  const lines = ['# Guide', '## Install', '### Windows', '### Mac', '## Use', '##### Too deep', '## Use']

  it('lists the levels asked for, nested from the shallowest that is there', () => {
    expect(renderToc(lines, options)).toEqual([
      '- [Install](#install)', '  - [Windows](#windows)', '  - [Mac](#mac)', '- [Use](#use)', '- [Use](#use-1)',
    ])
  })

  it('numbers the items of each level when asked', () => {
    expect(renderToc(lines, { ...options, ordered: true })).toEqual([
      '1. [Install](#install)', '   1. [Windows](#windows)', '   2. [Mac](#mac)', '2. [Use](#use)', '3. [Use](#use-1)',
    ])
  })

  it('is empty when there is nothing to list', () => {
    expect(renderToc(['# Only a title'], options)).toEqual([])
  })
})

describe('insertToc and updateToc', () => {
  const doc = ['# Guide', '', '## One', '', '## Two']

  it('writes a table of contents between marks at the cursor line', () => {
    const lines = [...doc.slice(0, 2), '', ...doc.slice(2)]

    expect(applied(lines, [insertToc(lines, { line: 2, character: 0 }, options)])).toEqual([
      '# Guide', '', '<!-- toc -->', '- [One](#one)', '- [Two](#two)', '<!-- /toc -->', '', '', '## One', '', '## Two',
    ].filter((_, index) => index !== 7))
  })

  it('keeps the table of contents current, in place, and says nothing is to be done when it is', () => {
    const lines = ['# Guide', '<!-- toc -->', '- [Old](#old)', '<!-- /toc -->', '## One', '## Two']
    const updated = applied(lines, [updateToc(lines, options)])

    expect(updated).toEqual(['# Guide', '<!-- toc -->', '- [One](#one)', '- [Two](#two)', '<!-- /toc -->', '## One', '## Two'])
    expect(updateToc(updated, options)).toBeUndefined()
  })

  it('does nothing without both marks, and a second insert updates instead of adding another', () => {
    expect(updateToc(['# A', '<!-- toc -->', '## B'], options)).toBeUndefined()
    const lines = ['<!-- toc -->', '<!-- /toc -->', '## B']

    expect(applied(lines, [insertToc(lines, { line: 2, character: 0 }, options)])).toEqual(['<!-- toc -->', '- [B](#b)', '<!-- /toc -->', '## B'])
  })

  it('empties the list when no heading is left to show', () => {
    const lines = ['<!-- toc -->', '- [Gone](#gone)', '<!-- /toc -->']

    expect(applied(lines, [updateToc(lines, options)])).toEqual(['<!-- toc -->', '<!-- /toc -->'])
  })
})

describe('section numbers', () => {
  const lines = ['# Title', '## A', '### A one', '### A two', '## B', '#### Deep']

  it('numbers the headings by level, not touching the title', () => {
    expect(applied(lines, addSectionNumbers(lines, options))).toEqual(['# Title', '## 1 A', '### 1.1 A one', '### 1.2 A two', '## 2 B', '#### 2.1.1 Deep'])
  })

  it('replaces numbers that are there, so running it again changes nothing', () => {
    const once = applied(lines, addSectionNumbers(lines, options))

    expect(addSectionNumbers(once, options)).toEqual([])
    expect(applied(['## 7 A', '## 9. B'], addSectionNumbers(['## 7 A', '## 9. B'], options))).toEqual(['## 1 A', '## 2 B'])
  })

  it('takes the numbers off', () => {
    const numbered = applied(lines, addSectionNumbers(lines, options))

    expect(applied(numbered, removeSectionNumbers(numbered, options))).toEqual(lines)
  })
})
