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
import type { ConnectionStore } from '../secret-storage'
import { type StorageSetupUI } from '../storage-setup'
import { type PlanPreviewUI, previewPlan } from './plan-preview.handler'
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
}

/** Everything the sync flow asks of the editor. Composes the first-run storage setup (which
 *  carries the target prompts, the connection picker and the GitHub Pages prompts), the plan
 *  preview, the output-subset picker, and the progress/report surface. */
export interface SyncUI extends StorageSetupUI, PlanPreviewUI {
  pickOutputs (choices: OutputChoice[]): Promise<number[] | undefined>
  withProgress<T> (title: string, task: (report: (step: ProgressStep) => void) => Promise<T>): Promise<T>
  report (lines: string[]): void
  status (message: string): void
  confirmForce (message: string): Promise<boolean>
}

/** What a sync needs: the engine, the stored connections and targets, the folder, an
 *  optional file scope, and the editor UI. */
export interface SyncDeps {
  engine:        EngineClient
  connections:   ConnectionStore
  targets:       TargetStore
  workspaceRoot: string
  scope?:        string[]
  ui:            SyncUI
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

  let session: SessionOpenResult
  try {
    session = await engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl: meta.baseUrl, edition: meta.edition, credential })
  } catch (error) {
    await ui.error(messageOf(error))

    return
  }

  // The engine refreshed an expired OAuth token and handed back a renewed pair: re-store it
  // so the next sync starts from the fresh access token.
  if (session.tokens) {
    await connections.add(meta, { kind: 'oauth', accessToken: session.tokens.accessToken, refreshToken: session.tokens.refreshToken ?? credential.refreshToken, clientId: credential.clientId })
  }

  try {
    targets.set(workspaceRoot, { space: output.space, parentPageId: output.parentPageId, parentTitle: '', titlePrefix: output.titlePrefix })
    await planAndExecute({ ...deps, sessionId: session.sessionId, output: index })
  } catch (error) {
    await ui.error(messageOf(error))
  } finally {
    try {
      await engine.request(SESSION_CLOSE_METHOD, { sessionId: session.sessionId })
    } catch {
      // Closing is best-effort.
    }
  }
}

async function planAndExecute (deps: SyncDeps & { sessionId: string; output: number }): Promise<void> {
  const { engine, sessionId, workspaceRoot, scope, output, ui } = deps

  const plan = await planOnce(engine, { sessionId, workspaceRoot, output, scope })
  if (!await previewPlan(plan, ui)) {
    return
  }

  const result = await execute(engine, plan.planId, false, ui)
  reportResult(result, ui)

  if ((plan.counts.conflict ?? 0) > 0) {
    const forced = await ui.confirmForce('Some pages changed on the platform since the last sync. Overwrite them?')
    if (forced) {
      const replan = await planOnce(engine, { sessionId, workspaceRoot, output, scope })
      reportResult(await execute(engine, replan.planId, true, ui), ui)
    }
  }
}

function planOnce (engine: EngineClient, params: { sessionId: string; workspaceRoot: string; output: number; scope?: string[] }): Promise<SyncPlanResult> {
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
function sameSite (a: string, b: string): boolean {
  return normaliseUrl(a) === normaliseUrl(b)
}

function normaliseUrl (url: string): string {
  return url.replace(/\/+$/, '').toLowerCase()
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export { summarizePlan } from './plan-preview.handler'
