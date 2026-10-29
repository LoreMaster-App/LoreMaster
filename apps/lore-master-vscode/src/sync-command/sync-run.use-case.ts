import type { EngineClient } from '../engine-process'
import {
  HOST_PROGRESS_METHOD,
  type Output,
  type ProgressParams,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  type SessionOpenResult,
  SYNC_EXECUTE_METHOD,
  SYNC_PLAN_METHOD,
  type SyncExecuteResult,
  type SyncPlanResult,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { type StorageSetupUI } from '../storage-setup'
import { type PlanPreviewUI, planDetails, summarizePlan } from './plan-preview.handler'
import type { TargetStore } from '../sync-target'

/** One progress step, as `withProgress` reports it. */
export interface ProgressStep {
  message: string
  done:    number
  total:   number
}

/** One configured output offered for a subset sync, with its index in the settings. */
export interface OutputChoice {
  index:  number
  output: Output
  /** What the engine calls it, and where it goes. */
  label:  string
  detail: string
}

/** Everything the sync flow asks of the editor. Composes the first-run storage setup (which
 *  carries the target prompts, the connection picker and the GitHub Pages prompts), the plan
 *  preview, the output-subset picker, and the progress/report surface. */
export interface SyncUI extends StorageSetupUI, PlanPreviewUI {
  pickOutputs (choices: OutputChoice[]): Promise<number[] | undefined>
  confirmForce (message: string): Promise<boolean>
  withProgress<T> (title: string, task: (report: (step: ProgressStep) => void) => Promise<T>): Promise<T>
  report (lines: string[]): void
  status (message: string): void
  /** Tells the user the sync can be undone; not every surface offers it (watch mode stays quiet). */
  offerRollback? (journalId: string): void
}

/** What a sync needs: the engine, the stored connections and targets, the folder, an
 *  optional file scope, the editor UI, and — for first-run setup — a way to add a connection
 *  inline when there is none. */
export interface SyncDeps {
  engine:         EngineClient
  connections:    ConnectionStore
  targets:        TargetStore
  workspaceRoot:  string
  scope?:         string[]
  /** Write every page again even when nothing changed. */
  rewrite?:       boolean
  ui:             SyncUI
  addConnection?: () => Promise<ConnectionMeta | undefined>
}

/** One configured Confluence output to sync, with its index in the settings' outputs. */
export interface ConfluenceOutputRef {
  output: Output
  index:  number
}

/**
 * Syncs one configured Confluence output: find the connection for its site, open a session,
 * plan, preview, execute under a progress bar, and report. A conflict offers a forced
 * re-run. The session is always closed. The output is already configured — the first-run
 * pickers live in the storage-setup flow.
 */
export async function syncConfluenceOutput (deps: SyncDeps, ref: ConfluenceOutputRef): Promise<void> {
  const { engine, connections, targets, workspaceRoot, ui } = deps
  const { output, index } = ref

  const meta = connections.list().find(connection => sameSite(connection.baseUrl, output.baseUrl))
  if (!meta) {
    await ui.error(`No connection for ${output.baseUrl}. Add it with "LoreMaster: Add Connection", then sync again.`)

    return
  }
  const credential = await connections.credential(meta.baseUrl)
  if (!credential) {
    await ui.error(`No stored credential for ${meta.baseUrl}. Add the connection again.`)

    return
  }

  let session: SessionOpenResult | undefined
  let planning: Promise<SyncPlanResult> | undefined
  // Connecting and planning start only once the user has chosen, and say what they are doing.
  const prepare = (): Promise<SyncPlanResult> => planning ??= ui.withProgress('LoreMaster: preparing the sync', async report => {
    report({ message: `connecting to ${meta.baseUrl}`, done: 0, total: 0 })
    session = await engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl: meta.baseUrl, edition: meta.edition, credential })
    targets.set(workspaceRoot, { space: output.space, parentPageId: output.parentPageId, parentTitle: '', titlePrefix: output.titlePrefix })
    report({ message: 'reading the workspace and checking each page', done: 0, total: 0 })

    return planOnce(engine, { sessionId: session.sessionId, workspaceRoot, output: index, scope: deps.scope, rewrite: deps.rewrite })
  })

  try {
    await runChosen(deps, prepare)
  } catch (error) {
    await ui.error(messageOf(error))
  } finally {
    if (session) {
      try {
        await engine.request(SESSION_CLOSE_METHOD, { sessionId: session.sessionId })
      } catch {
        // Closing is best-effort.
      }
    }
  }
}

/** The choice is on screen at once; the plan is made when it is needed. Conflicting pages are asked about before
 *  anything is written, never after. */
async function runChosen (deps: SyncDeps, prepare: () => Promise<SyncPlanResult>): Promise<void> {
  const { engine, ui } = deps

  for (;;) {
    const choice = await ui.choose()
    if (choice === 'cancel') {
      return
    }
    if (choice === 'run') {
      break
    }
    ui.writeDetails(planDetails(await prepare()))
  }

  const plan = await prepare()
  if ((plan.errors ?? []).length > 0) {
    ui.writeDetails(planDetails(plan))
    await ui.showErrors(plan.errors ?? [])

    return
  }
  const conflicts = plan.counts.conflict ?? 0
  const force = conflicts > 0 && await ui.confirmForce(`${conflicts} page(s) changed on the platform since the last sync. Overwrite them? (${summarizePlan(plan)})`)
  const result = await execute(engine, plan.planId, force, ui)
  reportResult(result, ui)
  offerRollback(result, ui)
}

function planOnce (engine: EngineClient, params: { sessionId: string; workspaceRoot: string; output: number; scope?: string[]; rewrite?: boolean }): Promise<SyncPlanResult> {
  return engine.request<SyncPlanResult>(SYNC_PLAN_METHOD, params)
}

function execute (engine: EngineClient, planId: string, force: boolean, ui: SyncUI): Promise<SyncExecuteResult> {
  return ui.withProgress('LoreMaster: syncing', async report => {
    const subscription = engine.onNotification(HOST_PROGRESS_METHOD, params => {
      const progress = params as ProgressParams
      if (progress.planId === planId) {
        report({ message: progress.message, done: progress.done, total: progress.total })
      }
    })
    try {
      return await engine.request<SyncExecuteResult>(SYNC_EXECUTE_METHOD, { planId, force })
    } finally {
      subscription.dispose()
    }
  })
}

function offerRollback (result: SyncExecuteResult, ui: SyncUI): void {
  if (result.journalId) {
    ui.offerRollback?.(result.journalId)
  }
}

function reportResult (result: SyncExecuteResult, ui: SyncUI): void {
  const pages = result.pages ?? []
  const lines = pages.map(page => `${page.outcome.padEnd(10)}${page.title}${page.error ? `  — ${page.error}` : ''}`)
  const warnings = result.warnings ?? []
  for (const warning of warnings) {
    lines.push(`warning: ${warning}`)
  }
  ui.report(lines)
  const failed = pages.filter(page => page.outcome === 'failed').length
  ui.status(failed > 0 ? `LoreMaster: synced with ${failed} failure(s)` : `LoreMaster: synced ${pages.length} page(s)`)
}

/** Two base URLs point at the same site when they match but for a trailing slash or case. */
/** Whether two Confluence base URLs name the same site (trailing slashes and case ignored). */
export function sameSite (a: string, b: string): boolean {
  return normaliseUrl(a) === normaliseUrl(b)
}

function normaliseUrl (url: string): string {
  return url.replace(/\/+$/, '').toLowerCase()
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export { summarizePlan } from './plan-preview.handler'
