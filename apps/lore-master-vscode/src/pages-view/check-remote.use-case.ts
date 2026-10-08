import type { EngineClient } from '../engine-process'
import {
  type Output,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  type SessionOpenResult,
  SYNC_PLAN_METHOD,
  type SyncPlanResult,
} from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { sameSite } from '../sync-command'
import type { RemoteCheck } from './page-tree.contract'

export interface CheckRemoteDeps {
  engine:        EngineClient
  connections:   ConnectionStore
  workspaceRoot: string
}

/** The result of asking the platform: what it said, or why it could not be asked. */
export type CheckRemoteOutcome = { ok: true; check: RemoteCheck } | { ok: false; reason: string }

/**
 * Asks the platform where an output's pages stand, writing nothing: finds the connection
 * for its site, opens a session, plans the sync (a plan changes nothing until it is
 * executed) and closes the session. The plan's per-file action is the remote half of the
 * Pages view's status.
 */
export async function checkRemote (deps: CheckRemoteDeps, output: Output, index: number): Promise<CheckRemoteOutcome> {
  const { engine, connections, workspaceRoot } = deps

  const meta = connections.list().find(connection => sameSite(connection.baseUrl, output.baseUrl))
  if (!meta) {
    return { ok: false, reason: `No connection for ${output.baseUrl}. Add it with "LoreMaster: Add Connection".` }
  }
  const credential = await connections.credential(meta.baseUrl)
  if (!credential) {
    return { ok: false, reason: `No stored credential for ${meta.baseUrl}. Add the connection again.` }
  }

  let session: SessionOpenResult
  try {
    session = await engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl: meta.baseUrl, edition: meta.edition, credential })
  } catch (error) {
    return { ok: false, reason: messageOf(error) }
  }

  try {
    const plan = await engine.request<SyncPlanResult>(SYNC_PLAN_METHOD, { sessionId: session.sessionId, workspaceRoot, output: index })
    const actions: Record<string, string> = {}
    const orphans: RemoteCheck['orphans'] = []
    for (const action of plan.actions) {
      if (action.kind === 'orphan') {
        orphans.push({ title: action.title, pageId: action.pageId, url: action.url })
      } else if (action.path !== undefined) {
        actions[action.path] = action.kind
      }
    }

    return { ok: true, check: { actions, orphans } }
  } catch (error) {
    return { ok: false, reason: messageOf(error) }
  } finally {
    try {
      await engine.request(SESSION_CLOSE_METHOD, { sessionId: session.sessionId })
    } catch {
      // Closing is best-effort.
    }
  }
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
