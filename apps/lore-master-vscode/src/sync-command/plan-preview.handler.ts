import type { SyncPlanResult } from '../engine-protocol'

/** The plan-preview prompts. */
export interface PlanPreviewUI {
  /** Offers Run / Show details / Cancel over a one-line summary, and, when `conflicts` pages were edited on the
   *  platform since the last sync, Run and overwrite them. */
  choose (summary: string, conflicts: number): Promise<'run' | 'force' | 'details' | 'cancel'>
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

/**
 * Shows the plan and asks whether to run it. A plan with errors cannot run: its details go
 * to the output channel and the errors are shown. Otherwise the summary is offered with
 * Run / Show details (writes the plan to the output channel, then asks again) / Cancel, plus
 * overwriting conflicting pages when there are any, so nothing is asked after the sync has run.
 * Returns how to execute, or 'cancel'.
 */
export async function previewPlan (plan: SyncPlanResult, ui: PlanPreviewUI): Promise<'run' | 'force' | 'cancel'> {
  if ((plan.errors ?? []).length > 0) {
    ui.writeDetails(planDetails(plan))
    await ui.showErrors(plan.errors ?? [])

    return 'cancel'
  }

  for (;;) {
    const choice = await ui.choose(summarizePlan(plan), plan.counts?.conflict ?? 0)
    if (choice !== 'details') {
      return choice
    }
    ui.writeDetails(planDetails(plan))
  }
}
