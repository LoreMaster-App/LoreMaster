import type { EngineClient } from '../engine-process'
import {
  HOST_PROGRESS_METHOD,
  type Output,
  type ProgressParams,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  SETTINGS_READ_METHOD,
  SETTINGS_SAVE_METHOD,
  type SessionOpenResult,
  type SettingsReadResult,
  SYNC_EXECUTE_METHOD,
  SYNC_PLAN_METHOD,
  type SyncExecuteResult,
  type SyncPlanResult,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { pickParentPage, pickSpace, resolveTitlePrefix, type SyncTargetUI, type TargetStore } from '../sync-target'
import { type PlanPreviewUI, previewPlan } from './plan-preview.handler'

/** One progress step, as `withProgress` reports it. */
export interface ProgressStep {
  message: string
  done:    number
  total:   number
}

/** Everything the sync flow asks of the editor. Composes the target and preview prompts. */
export interface SyncUI extends SyncTargetUI, PlanPreviewUI {
  pickConnection (connections: ConnectionMeta[]): Promise<ConnectionMeta | undefined>
  noConnections (): Promise<void>
  withProgress<T> (title: string, task: (report: (step: ProgressStep) => void) => Promise<T>): Promise<T>
  report (lines: string[]): void
  status (message: string): void
  confirmForce (message: string): Promise<boolean>
  error (message: string): Promise<void>
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

/**
 * Syncs a folder: choose a connection, open a session, make sure the folder has a target
 * (space, parent page, prefix — asking once and saving to .lore-master.yaml), plan, preview,
 * execute under a progress bar, and report. A conflict offers a forced re-run. The session
 * is always closed.
 */
export async function runSync (deps: SyncDeps): Promise<void> {
  const { engine, connections, ui } = deps

  const metas = connections.list()
  if (metas.length === 0) {
    await ui.noConnections()

    return
  }
  const meta = metas.length === 1 ? metas[0] : await ui.pickConnection(metas)
  if (!meta) {
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
    const output = await resolveOutput({ ...deps, sessionId: session.sessionId, meta })
    if (output === undefined) {
      return
    }
    await planAndExecute({ ...deps, sessionId: session.sessionId, output })
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

async function resolveOutput (deps: SyncDeps & { sessionId: string; meta: ConnectionMeta }): Promise<number | undefined> {
  const { engine, sessionId, workspaceRoot, meta, targets, ui } = deps

  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  const outputs = [...read.settings.outputs]
  let index = outputs.findIndex(output => output.baseUrl === meta.baseUrl)
  const existing = outputs[index]

  if (existing && existing.space !== '' && existing.parentPageId !== '' && existing.titlePrefix !== '') {
    targets.set(workspaceRoot, { space: existing.space, parentPageId: existing.parentPageId, parentTitle: '', titlePrefix: existing.titlePrefix })

    return index
  }

  const space = await pickSpace({ engine, sessionId, ui })
  if (!space) {
    return undefined
  }
  const parent = await pickParentPage({ engine, sessionId, space, ui })
  if (!parent) {
    return undefined
  }
  const titlePrefix = await resolveTitlePrefix({ existing: existing?.titlePrefix ?? '', parentTitle: parent.title, ui })
  if (titlePrefix === undefined) {
    return undefined
  }

  const output: Output = {
    platform:       'confluence',
    baseUrl:        meta.baseUrl,
    space:          space.key,
    parentPageId:   parent.pageId,
    titlePrefix,
    direction:      existing?.direction ?? 'push',
    content:        existing && existing.content.length > 0 ? existing.content : [{ type: 'markdown', roots: ['.'], template: '' }],
    mermaidMode:    existing?.mermaidMode ?? 'image',
    titleCollision: existing?.titleCollision ?? 'fail',
    linkMode:       existing?.linkMode ?? 'title',
  }
  if (index >= 0) {
    outputs[index] = output
  } else {
    outputs.push(output)
    index = outputs.length - 1
  }
  await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { version: read.settings.version || 1, outputs } })
  targets.set(workspaceRoot, { space: space.key, parentPageId: parent.pageId, parentTitle: parent.title, titlePrefix })

  return index
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
  return ui.withProgress('Lore Master: syncing', async report => {
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
  const lines = result.pages.map(page => `${page.outcome.padEnd(10)}${page.title}${page.error ? `  — ${page.error}` : ''}`)
  const warnings = result.warnings ?? []
  for (const warning of warnings) {
    lines.push(`warning: ${warning}`)
  }
  ui.report(lines)
  const failed = result.pages.filter(page => page.outcome === 'failed').length
  ui.status(failed > 0 ? `Lore Master: synced with ${failed} failure(s)` : `Lore Master: synced ${result.pages.length} page(s)`)
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export { summarizePlan } from './plan-preview.handler'
