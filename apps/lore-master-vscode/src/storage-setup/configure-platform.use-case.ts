import { blankOutput } from '../config-description'
import { type FieldDescriptor, type Output, type PlatformDescriptor, SESSION_CLOSE_METHOD, SESSION_OPEN_METHOD, type SessionOpenResult, type Space } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { pickParentPage, pickSpace } from '../sync-target'
import type { ConnectedSetupUI, StorageSetupDeps } from './storage-setup.contract'

/** What a lookup has chosen so far, for the fields that are filled from it. */
interface Chosen {
  space?: Space
  page?:  { pageId: string, title: string }
}

/**
 * Configures one storage the way its description says, field by field: the connection is chosen
 * (or added) and a session opened when the platform signs in; a field with a space lookup is picked
 * from the platform's spaces, one with a page lookup is browsed for in that space, and the others are
 * asked for, a field the lookup fills starting from what was chosen. Advanced fields keep their
 * defaults. Nothing is saved here. Returns undefined when the user cancels.
 */
export async function configurePlatform (deps: StorageSetupDeps, platform: PlatformDescriptor): Promise<Output | undefined> {
  const { engine, connections, ui, addConnection, description } = deps
  const output = blankOutput(description, platform) as unknown as Record<string, unknown>

  let meta: ConnectionMeta | undefined
  let session: SessionOpenResult | undefined
  if (platform.connection) {
    meta = await chooseConnection(connections, ui, addConnection)
    if (!meta) {
      return undefined
    }
    const credential = await connections.credential(meta.baseUrl)
    if (!credential) {
      await ui.error(`No stored credential for ${meta.baseUrl}. Add the connection again.`)

      return undefined
    }
    try {
      session = await engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl: meta.baseUrl, edition: meta.edition, credential })
    } catch (error) {
      await ui.error(messageOf(error))

      return undefined
    }
  }

  try {
    const chosen: Chosen = {}
    const asked = platform.fields.filter(each => each.advanced !== true)
    for (const field of asked) {
      const value = await valueFor({ deps, field, platform, meta, session, chosen })
      if (value === undefined) {
        return undefined
      }
      output[field.key] = value
    }

    return output as unknown as Output
  } catch (error) {
    await ui.error(messageOf(error))

    return undefined
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

interface ValueRequest {
  deps:     StorageSetupDeps
  field:    FieldDescriptor
  platform: PlatformDescriptor
  meta?:    ConnectionMeta
  session?: SessionOpenResult
  chosen:   Chosen
}

/** The value of one field: from the connection, a lookup, or the user. Undefined cancels the setup. */
async function valueFor ({ deps, field, platform, meta, session, chosen }: ValueRequest): Promise<string | undefined> {
  if (meta && platform.connection?.field === field.key) {
    return meta.baseUrl
  }
  if (session && field.lookup?.kind === 'space') {
    const space = await pickSpace({ engine: deps.engine, sessionId: session.sessionId, ui: deps.ui })
    chosen.space = space

    return space?.key
  }
  if (session && chosen.space && field.lookup?.kind === 'page') {
    const parent = await pickParentPage({ engine: deps.engine, sessionId: session.sessionId, space: chosen.space, ui: deps.ui })
    chosen.page = parent

    return parent?.pageId
  }
  const value = await deps.ui.promptField(field, startingValue(field, platform, chosen))

  return value?.trim()
}

/** What a field starts as: the property of a lookup's choice that the description says fills it. */
function startingValue (field: FieldDescriptor, platform: PlatformDescriptor, chosen: Chosen): string | undefined {
  for (const source of platform.fields) {
    const fill = source.lookup?.fills?.find(each => each.field === field.key)
    if (fill) {
      return fill.from === 'id' ? chosen.page?.pageId : chosen.page?.title
    }
  }

  return undefined
}

/** Picks the connection to use: the only one, a chosen one, or - when there is none - a freshly
 *  added one (if the caller provided an Add Connection flow), rather than stopping. */
async function chooseConnection (connections: ConnectionStore, ui: ConnectedSetupUI, addConnection?: () => Promise<ConnectionMeta | undefined>): Promise<ConnectionMeta | undefined> {
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
