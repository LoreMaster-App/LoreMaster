import type { EngineClient } from '../engine-process'
import {
  type Page,
  PAGE_SEARCH_METHOD,
  type PagesResult,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  type SessionOpenResult,
  type Space,
  SPACE_LIST_METHOD,
  type SpaceListResult,
} from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'

/** Looks up the spaces and pages of a stored connection while the form is open. */
export interface StorageLookup {
  listSpaces (baseUrl: string): Promise<Space[]>
  searchPages (baseUrl: string, spaceKey: string, query: string): Promise<Page[]>
  dispose (): Promise<void>
}

/**
 * One engine session per site, opened on first use with the stored credential and closed when
 * the form closes. A session the engine no longer knows is opened again once.
 */
export function createStorageLookup (deps: { engine: EngineClient; connections: ConnectionStore }): StorageLookup {
  const sessions = new Map<string, string>()

  async function open (baseUrl: string): Promise<string> {
    const meta = deps.connections.list().find(connection => connection.baseUrl === baseUrl)
    const credential = await deps.connections.credential(baseUrl)
    if (!meta || !credential) {
      throw new Error(`There is no stored credential for ${baseUrl}. Add the connection again.`)
    }
    const session = await deps.engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl, edition: meta.edition, credential })
    sessions.set(baseUrl, session.sessionId)

    return session.sessionId
  }

  async function withSession<T> (baseUrl: string, call: (sessionId: string) => Promise<T>): Promise<T> {
    const known = sessions.get(baseUrl)
    if (known) {
      try {
        return await call(known)
      } catch {
        sessions.delete(baseUrl)
      }
    }

    return call(await open(baseUrl))
  }

  return {
    listSpaces: baseUrl => withSession(baseUrl, async sessionId => {
      const result = await deps.engine.request<SpaceListResult>(SPACE_LIST_METHOD, { sessionId })

      return result.spaces
    }),
    searchPages: (baseUrl, spaceKey, query) => withSession(baseUrl, async sessionId => {
      const result = await deps.engine.request<PagesResult>(PAGE_SEARCH_METHOD, { sessionId, spaceKey, query, limit: 25 })

      return result.pages
    }),
    async dispose () {
      for (const sessionId of sessions.values()) {
        try {
          await deps.engine.request(SESSION_CLOSE_METHOD, { sessionId })
        } catch {
          // Closing is best-effort.
        }
      }
      sessions.clear()
    },
  }
}
