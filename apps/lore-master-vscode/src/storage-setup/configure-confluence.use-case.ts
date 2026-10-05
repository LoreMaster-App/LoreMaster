import type { EngineClient } from '../engine-process'
import { type Output, SESSION_CLOSE_METHOD, SESSION_OPEN_METHOD, type SessionOpenResult } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { pickParentPage, pickSpace, resolveTitlePrefix } from '../sync-target'
import type { ConfluenceSetupUI } from './set-up-storages.use-case'

/** What configuring a Confluence output needs. `addConnection`, when given, lets the flow
 *  start the Add Connection wizard inline when there is no connection yet, instead of
 *  stopping with an instruction. */
export interface ConfigureConfluenceDeps {
  engine:         EngineClient
  connections:    ConnectionStore
  ui:             ConfluenceSetupUI
  addConnection?: () => Promise<ConnectionMeta | undefined>
}

/** Configures one Confluence output: choose (or add) a connection, open a session to pick
 *  the space, parent page and title prefix, and build the output (not saved here). Returns
 *  undefined when the user cancels. */
export async function configureConfluence (deps: ConfigureConfluenceDeps): Promise<Output | undefined> {
  const { engine, connections, ui, addConnection } = deps

  const meta = await chooseConnection(connections, ui, addConnection)
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

/** Picks the connection to use: the only one, a chosen one, or — when there is none — a
 *  freshly added one (if the caller provided an Add Connection flow), rather than stopping. */
async function chooseConnection (connections: ConnectionStore, ui: ConfluenceSetupUI, addConnection?: () => Promise<ConnectionMeta | undefined>): Promise<ConnectionMeta | undefined> {
  const metas = connections.list()
  if (metas.length === 0) {
    if (!addConnection) {
      await ui.noConnections()

      return undefined
    }

    return addConnection()
  }

  return metas.length === 1 ? metas[0] : ui.pickConnection(metas)
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
