import type { SyncPlanResult } from '../engine-protocol'

/** The plan-preview prompts. */
export interface PlanPreviewUI {
  /** Offers Run / Show details / Cancel the moment a sync starts, before anything is read. */
  choose (): Promise<'run' | 'details' | 'cancel'>
  /** Writes the full plan to the output channel. */
  writeDetails (lines: string[]): void
  /** Shows that the plan cannot run, with its errors. */
  showErrors (errors: string[]): Promise<void>
}

/** A one-line summary: the non-zero action counts, e.g. "2 create, 1 update". */
export function summarizePlan (plan: SyncPlanResult): string {
  const parts = Object.entries(plan.counts ?? {})
    .filter(([, count]) => count > 0)
    .map(([kind, count]) => `${count} ${kind}`)

  return parts.length > 0 ? parts.join(', ') : 'nothing to do'
}

/** The plan as lines for the output channel: one per action, then warnings and errors. */
export function planDetails (plan: SyncPlanResult): string[] {
  const lines = (plan.actions ?? []).map(action => {
    const where = action.path ? `  (${action.path})` : ''
    const why = action.reason ? `  — ${action.reason}` : ''

    return `${action.kind.padEnd(13)}${action.title}${where}${why}`
  })
  const warnings = plan.warnings ?? []
  for (const warning of warnings) {
    lines.push(`warning: ${warning}`)
  }
  const errors = plan.errors ?? []
  for (const error of errors) {
    lines.push(`error: ${error}`)
  }

  return lines
}
