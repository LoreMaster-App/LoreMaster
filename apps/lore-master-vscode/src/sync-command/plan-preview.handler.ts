import type { SyncPlanResult } from '../engine-protocol'

/** The plan-preview prompts. */
export interface PlanPreviewUI {
  /** Offers Run / Show details / Cancel over a one-line summary. */
  choose (summary: string): Promise<'run' | 'details' | 'cancel'>
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
 * Run / Show details (writes the plan to the output channel, then asks again) / Cancel.
 * Returns true to execute.
 */
export async function previewPlan (plan: SyncPlanResult, ui: PlanPreviewUI): Promise<boolean> {
  if ((plan.errors ?? []).length > 0) {
    ui.writeDetails(planDetails(plan))
    await ui.showErrors(plan.errors ?? [])

    return false
  }

  for (;;) {
    const choice = await ui.choose(summarizePlan(plan))
    if (choice === 'run') {
      return true
    }
    if (choice === 'cancel') {
      return false
    }
    ui.writeDetails(planDetails(plan))
  }
}
