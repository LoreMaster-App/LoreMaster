import { DIAGRAM_CHECK_METHOD, type DiagramCheckResult } from '../engine-protocol'
import type { CheckedLanguage, DiagramFence, DiagramFinding } from './diagram-fence.contract'

/** The one thing checking needs from the engine client: to make a request. */
export interface CheckEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** What the engine said about one diagram's source. */
export interface CheckOutcome {
  problems:     { line?: number, column?: number, message: string }[]
  unavailable?: string
}

/** Known answers by language and source, so an unchanged diagram is not asked about again. */
export type CheckCache = Map<string, CheckOutcome>

const SEPARATOR = '\u{0}'
const keyOf = (fence: DiagramFence): string => `${fence.language}${SEPARATOR}${fence.source}`

function placed (fence: DiagramFence, problem: { line?: number, column?: number, message: string }): DiagramFinding {
  const lines = fence.source.split('\n').length
  // A problem with no line sits on the opening fence; one past the end is held to the last line of the source.
  const inSource = problem.line === undefined || problem.line <= 0 ? undefined : Math.min(problem.line, lines) - 1

  return {
    language: fence.language,
    line:     inSource === undefined ? Math.max(fence.firstLine - 1, 0) : fence.firstLine + inSource,
    column:   inSource === undefined ? 0 : Math.max((problem.column ?? 1) - 1, 0),
    message:  problem.message,
  }
}

/**
 * What is wrong with the diagrams of a document, placed on the document's lines. A diagram already in `cache` is not asked about
 * again; one of a language not in `ask` that is not in the cache is left unchecked (Mermaid is only checked when the file is
 * opened or saved, since it starts a browser). `unavailable` is why some diagram could not be checked, for the output channel; such a
 * diagram is not remembered, so the next check tries it again.
 */
export async function checkDiagrams (
  engine: CheckEngine, fences: DiagramFence[], cache: CheckCache, ask: readonly CheckedLanguage[],
): Promise<{ findings: DiagramFinding[], unavailable: string[] }> {
  const missing = fences.filter(fence => !cache.has(keyOf(fence)) && ask.includes(fence.language))
  const seen = new Set<string>()
  const wanted = missing.filter(fence => {
    const key = keyOf(fence)

    return seen.has(key) ? false : Boolean(seen.add(key))
  })
  const unavailable: string[] = []
  if (wanted.length > 0) {
    const { results } = await engine.request<DiagramCheckResult>(DIAGRAM_CHECK_METHOD, { diagrams: wanted.map(fence => ({ language: fence.language, source: fence.source })) })
    for (const [index, fence] of wanted.entries()) {
      const outcome = results[index]
      if (outcome === undefined) { continue }
      if (outcome.unavailable === undefined) {
        cache.set(keyOf(fence), outcome)
      } else if (!unavailable.includes(outcome.unavailable)) {
        unavailable.push(outcome.unavailable)
      }
    }
  }

  const findings: DiagramFinding[] = []
  for (const fence of fences) {
    const known = cache.get(keyOf(fence))?.problems ?? []
    for (const problem of known) {
      findings.push(placed(fence, problem))
    }
  }

  return { findings, unavailable }
}
