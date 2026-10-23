import type { CheckedLanguage, DiagramFence } from './diagram-fence.contract'

const OPENING = /^ {0,3}(`{3,}|~{3,})\s*([^\s`]*)/
const CHECKED = new Set<string>(['mermaid', 'd2'])

/** Whether a line closes a fence opened with `marker`: the same character, at least as long, nothing after it. */
function closes (line: string, marker: string): boolean {
  const match = /^ {0,3}(`{3,}|~{3,})\s*$/.exec(line)

  return match !== null && match[1][0] === marker[0] && match[1].length >= marker.length
}

/**
 * The Mermaid and D2 fences of a Markdown document, in order, with the line each one's source starts on. Other fences are skipped
 * whole, so a ```mermaid inside a longer ````markdown block is not mistaken for a diagram. A fence never closed runs to the end.
 */
export function locateDiagramFences (text: string): DiagramFence[] {
  const lines = text.split(/\r?\n/)
  const fences: DiagramFence[] = []
  for (let index = 0; index < lines.length; index++) {
    const opening = OPENING.exec(lines[index])
    if (opening === null) { continue }
    const [, marker, info] = opening
    let end = index + 1
    while (end < lines.length && !closes(lines[end], marker)) { end++ }
    const language = info.toLowerCase()
    if (CHECKED.has(language)) {
      fences.push({ language: language as CheckedLanguage, source: lines.slice(index + 1, end).join('\n'), firstLine: index + 1 })
    }
    index = end
  }

  return fences
}
