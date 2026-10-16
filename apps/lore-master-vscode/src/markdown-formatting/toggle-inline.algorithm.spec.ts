import type { EditResult, TextSelection } from '../text-edits'
import { toggleInline, type InlineMarker } from './toggle-inline.algorithm'

/** Applies a result to the text the way an editor would, and says where the cursor is, as `[` and `]` (or `|`). */
function run (source: string, marker: InlineMarker): string {
  const text = source.replace('[', '').replace(']', '').replace('|', '')
  const lines = text.split('\n')
  const selection = selectionOf(source)
  const result = toggleInline(lines, selection, marker)

  return show(lines, result)
}

function selectionOf (source: string): TextSelection {
  const open = source.indexOf('[')
  const close = source.indexOf(']') - 1
  const bar = source.indexOf('|')
  const at = (offset: number): { line: number, character: number } => {
    const before = source.replace('[', '').replace(']', '').replace('|', '').slice(0, offset).split('\n')

    return { line: before.length - 1, character: before.at(-1)?.length ?? 0 }
  }
  if (bar !== -1) {
    return { anchor: at(bar), active: at(bar) }
  }

  return { anchor: at(open), active: at(close) }
}

function offsetOf (position: { line: number, character: number }, from: string[]): number {
  return from.slice(0, position.line).reduce((sum, line) => sum + line.length + 1, 0) + position.character
}

function show (lines: string[], result: EditResult): string {
  let text = lines.join('\n')
  const [edit] = result.edits
  text = text.slice(0, offsetOf(edit.range.start, lines)) + edit.text + text.slice(offsetOf(edit.range.end, lines))
  const next = text.split('\n')
  const anchor = offsetOf(result.selection.anchor, next)
  const active = offsetOf(result.selection.active, next)

  return anchor === active ? `${text.slice(0, anchor)}|${text.slice(anchor)}` : `${text.slice(0, anchor)}[${text.slice(anchor, active)}]${text.slice(active)}`
}

describe('toggleInline', () => {
  it('wraps a selection and keeps it selected', () => {
    expect(run('say [hello] there', '**')).toBe('say **[hello]** there')
    expect(run('say [hello] there', '*')).toBe('say *[hello]* there')
    expect(run('say [hello] there', '~~')).toBe('say ~~[hello]~~ there')
    expect(run('say [hello] there', '`')).toBe('say `[hello]` there')
  })

  it('takes formatting off again, whether the markers are selected or outside', () => {
    expect(run('say **[hello]** there', '**')).toBe('say [hello] there')
    expect(run('say [**hello**] there', '**')).toBe('say [hello] there')
  })

  it('works on the word at the cursor and keeps the cursor where it was in the word', () => {
    expect(run('say hel|lo there', '**')).toBe('say **hel|lo** there')
    expect(run('say **hel|lo** there', '**')).toBe('say hel|lo there')
  })

  it('leaves an empty pair to type between when the cursor is not in a word', () => {
    expect(run('say | there', '**')).toBe('say **|** there')
    expect(run('say |', '`')).toBe('say `|`')
  })

  it('keeps spaces at the edges of a selection outside the markers', () => {
    expect(run('say [ hello ] there', '**')).toBe('say  **[hello]**  there')
  })

  it('lets italic and bold sit together without disturbing each other', () => {
    expect(run('[**hello**]', '*')).toBe('*[**hello**]*')
    expect(run('***[hello]***', '*')).toBe('**[hello]**')
    expect(run('***[hello]***', '**')).toBe('*[hello]*')
    expect(run('*[hello]*', '**')).toBe('***[hello]***')
  })

  it('works across lines', () => {
    expect(run('[one\ntwo]', '**')).toBe('**[one\ntwo]**')
  })

  it('does not treat unicode letters as the edge of a word', () => {
    expect(run('un café|!', '**')).toBe('un **café|**!')
  })
})
