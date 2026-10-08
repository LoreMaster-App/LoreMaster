import type { GeneratorRun, GeneratorsRunResult } from '../engine-protocol'

/** The generators that could not run at all, as opposed to ones that ran with warnings. */
export function failedRuns (result: GeneratorsRunResult): GeneratorRun[] {
  return result.runs.filter(run => run.error !== undefined && run.error !== '')
}

/** What to ask before syncing when some generators failed: their pages are missing or stale. */
export function confirmationAfterFailures (failed: readonly GeneratorRun[]): string {
  const names = failed.map(run => run.type).join(', ')

  return failed.length === 1
    ? `The ${names} generator failed, so its pages may be missing or out of date. Sync anyway?`
    : `${failed.length} generators failed (${names}), so their pages may be missing or out of date. Sync anyway?`
}
