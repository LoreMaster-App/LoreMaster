import type { EngineClient } from '../engine-process'
import { type Output, SESSION_CLOSE_METHOD, SESSION_OPEN_METHOD, type SessionOpenResult } from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { pickParentPage, pickSpace, resolveTitlePrefix } from '../sync-target'
import type { ConfluenceSetupUI } from './set-up-storages.use-case'

/** Configures one Confluence output: choose a connection, open a session to pick the space,
 *  parent page and title prefix, and build the output (not saved here). Returns undefined
 *  when the user cancels or there is no connection to use. */
export async function configureConfluence (deps: { engine: EngineClient; connections: ConnectionStore; ui: ConfluenceSetupUI }): Promise<Output | undefined> {
  const { engine, connections, ui } = deps

  const metas = connections.list()
  if (metas.length === 0) {
    await ui.noConnections()

    return undefined
  }
  const meta = metas.length === 1 ? metas[0] : await ui.pickConnection(metas)
  if (!meta) {
    return undefined
  }
  const credential = await connections.credential(meta.baseUrl)
  if (!credential) {
    await ui.error(`No stored credential for ${meta.baseUrl}. Add the connection again.`)

    return undefined
  }

  let session: SessionOpenResult
  try {
    session = await engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl: meta.baseUrl, edition: meta.edition, credential })
  } catch (error) {
    await ui.error(messageOf(error))

    return undefined
  }

  try {
    const space = await pickSpace({ engine, sessionId: session.sessionId, ui })
    if (!space) {
      return undefined
    }
    const parent = await pickParentPage({ engine, sessionId: session.sessionId, space, ui })
    if (!parent) {
      return undefined
    }
    const titlePrefix = await resolveTitlePrefix({ existing: '', parentTitle: parent.title, ui })
    if (titlePrefix === undefined) {
      return undefined
    }

    return {
      platform:       'confluence',
      baseUrl:        meta.baseUrl,
      space:          space.key,
      parentPageId:   parent.pageId,
      titlePrefix,
      direction:      'to-platform',
      content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
      mermaidMode:    'image',
      titleCollision: 'fail',
      linkMode:       'title',
    }
  } catch (error) {
    await ui.error(messageOf(error))

    return undefined
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
