import { linesOf, replaceLines, type EditResult, type TextSelection } from '../text-edits'

const FENCE = /^\s*(?:```|~~~)/

/**
 * Fences the lines as a code block, or takes the fences off when the selection is a fenced block or lies inside
 * one. With nothing selected on an empty line it opens an empty block and leaves the cursor in it.
 */
export function toggleCodeBlock (lines: readonly string[], selection: TextSelection): EditResult | undefined {
  const { from, to } = linesOf(selection)

  const selectedFences = to > from && FENCE.test(lines[from]) && FENCE.test(lines[to])
  if (selectedFences) {
    const inner = lines.slice(from + 1, to)

    return replaceLines(lines, from, to, inner, { anchor: { line: from, character: 0 }, active: { line: from + Math.max(inner.length - 1, 0), character: inner.at(-1)?.length ?? 0 } })
  }

  const surrounded = from > 0 && to < lines.length - 1 && FENCE.test(lines[from - 1]) && FENCE.test(lines[to + 1])
  if (surrounded) {
    const inner = lines.slice(from, to + 1)

    return replaceLines(lines, from - 1, to + 1, inner, { anchor: { line: from - 1, character: 0 }, active: { line: from - 1 + inner.length - 1, character: inner.at(-1)?.length ?? 0 } })
  }

  const body = lines.slice(from, to + 1)
  const emptyCursor = body.length === 1 && body[0].trim() === ''
  const indent = /^\s*/.exec(body[0])?.[0] ?? ''
  const wrapped = [`${indent}\`\`\``, ...body, `${indent}\`\`\``]
  if (emptyCursor) {
    return replaceLines(lines, from, to, wrapped, { anchor: { line: from + 1, character: 0 }, active: { line: from + 1, character: 0 } })
  }

  return replaceLines(lines, from, to, wrapped, { anchor: { line: from + 1, character: 0 }, active: { line: to + 1, character: body.at(-1)?.length ?? 0 } })
}
