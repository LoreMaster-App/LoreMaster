import { replaceText, toOffset, type EditResult, type TextSelection } from '../text-edits'

/** The inline formatting a command can switch on or off. */
export type InlineMarker = '**' | '*' | '~~' | '`'

const WORD_CHARACTER = /[\p{L}\p{N}_]/u

/**
 * Whether a run of marker characters around some text means the formatting is on. A run of `*` is italic when
 * it is odd (`*a*`, `***a***`) and bold when it is two or more; the other markers are on from their own length.
 */
function formattingIsOn (marker: InlineMarker, run: number): boolean {
  switch (marker) {
    case '*': {
      return run % 2 === 1
    }
    case '**':
    case '~~': {
      return run >= 2
    }
    default: {
      return run >= 1
    }
  }
}

function runBefore (text: string, offset: number, character: string): number {
  let run = 0
  while (offset > run && text[offset - run - 1] === character) {
    run += 1
  }

  return run
}

function runAfter (text: string, offset: number, character: string): number {
  let run = 0
  while (offset + run < text.length && text[offset + run] === character) {
    run += 1
  }

  return run
}

/** The word the cursor is in or touches, if there is one. */
function wordAround (text: string, offset: number): [number, number] | undefined {
  let start = offset
  while (start > 0 && WORD_CHARACTER.test(text[start - 1])) {
    start -= 1
  }
  let end = offset
  while (end < text.length && WORD_CHARACTER.test(text[end])) {
    end += 1
  }

  return start === end ? undefined : [start, end]
}

/**
 * Switches inline formatting on the selected text, or on the word at the cursor, or leaves an empty pair of
 * markers for typing between. Formatting that is already on is taken off, whether the markers were selected
 * with the text or sit right outside it, and one kind of formatting never disturbs another: italic on bold
 * text makes `***text***`, and taking it off again makes `**text**`.
 */
export function toggleInline (lines: readonly string[], selection: TextSelection, marker: InlineMarker): EditResult {
  const text = lines.join('\n')
  const first = Math.min(toOffset(lines, selection.anchor), toOffset(lines, selection.active))
  const last = Math.max(toOffset(lines, selection.anchor), toOffset(lines, selection.active))
  const cursorOnly = first === last
  const length = marker.length

  let start = first
  let end = last
  if (cursorOnly) {
    const word = wordAround(text, first)
    if (!word) {
      return replaceText(lines, first, first, marker + marker, first + length, first + length)
    }
    [start, end] = word
  } else {
    while (start < end && /\s/.test(text[start])) {
      start += 1
    }
    while (end > start && /\s/.test(text[end - 1])) {
      end -= 1
    }
    if (start === end) {
      return replaceText(lines, first, first, marker + marker, first + length, first + length)
    }
  }

  const inner = text.slice(start, end)
  const character = marker[0]
  const leading = runAfter(inner, 0, character)
  const trailing = runBefore(inner, inner.length, character)
  const markersSelected = leading < inner.length && inner.length >= 2 * length && formattingIsOn(marker, Math.min(leading, trailing))
  const markersOutside = formattingIsOn(marker, Math.min(runBefore(text, start, character), runAfter(text, end, character)))

  // A lone cursor keeps its place in the word; a selection ends up around the same text.
  const place = (from: number, to: number, shift: number): [number, number] => (cursorOnly ? [first + shift, first + shift] : [from, to])

  if (markersSelected) {
    const unwrapped = inner.slice(length, inner.length - length)
    const [anchor, active] = place(start, start + unwrapped.length, -length)

    return replaceText(lines, start, end, unwrapped, anchor, active)
  }
  if (markersOutside) {
    const [anchor, active] = place(start - length, end - length, -length)

    return replaceText(lines, start - length, end + length, inner, anchor, active)
  }
  const [anchor, active] = place(start + length, end + length, length)

  return replaceText(lines, start, end, marker + inner + marker, anchor, active)
}
