import { describeConfig } from '../config-description'
import {
  type ConnectionDescriptor,
  type Credential,
  type CredentialKindDescriptor,
  type EditionDetectResult,
  type PlatformDescriptor,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  type SessionOpenResult,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { createConnectionUI } from './connection-ui.client'

/** The command id contributed in package.json. */
export const ADD_CONNECTION_COMMAND = 'loreMaster.addConnection'

/** The one thing the flow needs from the engine client: to make a request. */
export interface EngineRequester {
  request<R>(method: string, params?: unknown): Promise<R>
}

/** What the address prompt says, from the platform's description of its address field. */
export interface SitePrompt {
  platform:     string
  label:        string
  placeholder?: string
  help?:        string
}

/** The prompts the flow drives, abstracted so it tests without the `vscode` module. */
export interface ConnectionUI {
  /** Which platform to connect to, asked only when the engine describes more than one that signs in. */
  pickPlatform (platforms: PlatformDescriptor[]): Promise<PlatformDescriptor | undefined>
  promptBaseUrl (site: SitePrompt): Promise<string | undefined>
  pickAuthMethod (methods: CredentialKindDescriptor[]): Promise<CredentialKindDescriptor | undefined>
  promptCredential (method: CredentialKindDescriptor): Promise<Credential | undefined>
  showError (message: string): Promise<void>
  showConnected (meta: ConnectionMeta): Promise<void>
}

/**
 * The sign-in methods a platform's description lists for an edition, in the order to offer them. The engine makes the final
 * ruling (and reports why if the pairing is refused); this only narrows the list so Cloud is not asked for a PAT, etc.
 * A method that names no editions is offered for every one.
 */
export function authMethodsFor (connection: ConnectionDescriptor, edition: string): CredentialKindDescriptor[] {
  return connection.kinds.filter(kind => kind.editions === undefined || kind.editions.length === 0 || kind.editions.includes(edition))
}

/**
 * Walks the user through connecting to a platform as the engine describes it: ask for the site's address, detect the edition,
 * offer only the sign-in methods it accepts, collect what each asks for, verify it by opening a throwaway session, and on
 * success store the credential and metadata. The platform, its address field and its sign-in methods all come from the engine's
 * description, so another platform that signs in is connected the same way.
 *
 * A cancelled prompt ends the flow quietly. A failure from the engine — above all a refused credential — is shown verbatim, so
 * the message the user sees is the engine's. Returns the stored connection, or `undefined` if cancelled or unverified.
 */
export async function setUpConnection (deps: { engine: EngineRequester; store: ConnectionStore; ui: ConnectionUI }): Promise<ConnectionMeta | undefined> {
  const { engine, store, ui } = deps

  const platform = await choosePlatform(engine, ui)
  const connection = platform?.connection
  if (!platform || !connection) {
    return undefined
  }
  const address = platform.fields.find(field => field.key === connection.field)
  const baseUrl = await ui.promptBaseUrl({ platform: platform.label, label: address?.label ?? 'Site address', placeholder: address?.placeholder, help: address?.help })
  if (!baseUrl) {
    return undefined
  }

  let detected: EditionDetectResult = { baseUrl, edition: 'cloud' }
  if (connection.detectMethod) {
    try {
      detected = await engine.request<EditionDetectResult>(connection.detectMethod, { baseUrl })
    } catch (error) {
      await ui.showError(messageOf(error))

      return undefined
    }
  }

  const methods = authMethodsFor(connection, detected.edition)
  if (methods.length === 0) {
    await ui.showError(`${platform.label} has no sign-in method for a ${detected.edition} site.`)

    return undefined
  }
  const method = methods.length === 1 ? methods[0] : await ui.pickAuthMethod(methods)
  if (!method) {
    return undefined
  }

  const credential = await ui.promptCredential(method)
  if (!credential) {
    return undefined
  }

  let session: SessionOpenResult
  try {
    session = await engine.request<SessionOpenResult>(SESSION_OPEN_METHOD, { baseUrl: detected.baseUrl, edition: detected.edition, credential })
  } catch (error) {
    await ui.showError(messageOf(error))

    return undefined
  }

  // The verify session is not reused — the engine may restart before a sync — so close it.
  try {
    await engine.request(SESSION_CLOSE_METHOD, { sessionId: session.sessionId })
  } catch {
    // Closing is best-effort; a lost session costs nothing.
  }

  const meta: ConnectionMeta = {
    baseUrl:     session.baseUrl,
    edition:     session.edition,
    displayName: session.user.displayName,
    user:        session.user.username ?? session.user.accountId ?? '',
  }
  await store.add(meta, credential)
  await ui.showConnected(meta)

  return meta
}

/** The platform to connect to: the only one that signs in, or the user's pick when the engine describes several. */
async function choosePlatform (engine: EngineRequester, ui: ConnectionUI): Promise<PlatformDescriptor | undefined> {
  const description = await describeConfig(engine)
  const connecting = description.platforms.filter(platform => platform.connection !== undefined)

  return connecting.length > 1 ? ui.pickPlatform(connecting) : connecting[0]
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

/** Runs the Add Connection wizard with the default editor UI. A convenience for callers that
 *  just need "let the user add a connection now" (e.g. first-run sync with no connection). */
export function connectPlatform (deps: { engine: EngineRequester; store: ConnectionStore }): Promise<ConnectionMeta | undefined> {
  return setUpConnection({ engine: deps.engine, store: deps.store, ui: createConnectionUI() })
}
