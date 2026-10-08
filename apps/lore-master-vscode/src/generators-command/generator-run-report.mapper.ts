import type { GeneratorRun, GeneratorsRunResult } from '../engine-protocol'

/** What a run of generators did, ready to show: one headline and the lines behind it. */
export interface GeneratorRunReport {
  headline:    string
  /** One line per generator, each followed by its warnings and errors. */
  lines:       string[]
  /** True when anything went wrong or was left alone, so the lines are worth showing. */
  hasProblems: boolean
}

const plural = (count: number, noun: string): string => `${count} ${noun}${count === 1 ? '' : 's'}`

/** The line for one generator: its type, where it writes, and what it did there. */
function describeRun (run: GeneratorRun): string[] {
  const label = `${run.type} → ${run.output}`
  if (run.error) {
    return [`${label}: failed`, `  ✗ ${run.error}`]
  }
  const counts = [
    `${plural(run.written?.length ?? 0, 'page')} written`,
    `${run.unchanged?.length ?? 0} unchanged`,
    `${run.removed?.length ?? 0} removed`,
  ].join(', ')

  return [`${label}: ${counts}`, ...(run.warnings ?? []).map(warning => `  ! ${warning}`)]
}

/**
 * Summarises the engine's answer. Written, unchanged and removed are counted across all the
 * generators for the headline; a generator that failed, and every warning (an input skipped, a
 * file left alone because it was not generated), makes the report a problem report.
 */
export function summariseRun (result: GeneratorsRunResult): GeneratorRunReport {
  if (result.runs.length === 0) {
    return { headline: 'LoreMaster: there are no generators to run.', lines: [], hasProblems: false }
  }

  let written = 0
  let unchanged = 0
  let removed = 0
  let failed = 0
  let warnings = 0
  for (const run of result.runs) {
    written += run.written?.length ?? 0
    unchanged += run.unchanged?.length ?? 0
    removed += run.removed?.length ?? 0
    warnings += run.warnings?.length ?? 0
    if (run.error) {
      failed++
    }
  }

  const parts = [`${plural(written, 'page')} written`, `${unchanged} unchanged`, `${removed} removed`]
  if (failed > 0) {
    parts.push(`${failed} failed`)
  }
  const headline = `LoreMaster: ${parts.join(', ')} by ${plural(result.runs.length, 'generator')}.`

  return { headline, lines: result.runs.flatMap(run => describeRun(run)), hasProblems: failed > 0 || warnings > 0 }
}
