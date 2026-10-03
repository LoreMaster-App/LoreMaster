import {
  type Credential,
  type CredentialKind,
  type Edition,
  EDITION_DETECT_METHOD,
  type EditionDetectResult,
  SESSION_CLOSE_METHOD,
  SESSION_OPEN_METHOD,
  type SessionOpenResult,
} from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'

/** The command id contributed in package.json. */
export const ADD_CONNECTION_COMMAND = 'loreMaster.addConnection'

/** A sign-in method the user can pick. Same kinds as {@link Credential}. */
export type AuthMethod = CredentialKind

/** The one thing the flow needs from the engine client: to make a request. */
export interface EngineRequester {
  request<R>(method: string, params?: unknown): Promise<R>
}

/** The prompts the flow drives, abstracted so it tests without the `vscode` module. */
export interface ConnectionUI {
  promptBaseUrl (): Promise<string | undefined>
  pickAuthMethod (methods: AuthMethod[]): Promise<AuthMethod | undefined>
  promptCredential (method: AuthMethod): Promise<Credential | undefined>
  showError (message: string): Promise<void>
  showConnected (meta: ConnectionMeta): Promise<void>
}

/** The sign-in methods an edition accepts, in the order to offer them. The engine makes
 *  the final ruling (and reports why if the pairing is refused); this only narrows the
 *  list so Cloud is not asked for a PAT, etc. */
export function authMethodsFor (edition: Edition): AuthMethod[] {
  switch (edition) {
    case 'cloud': {
      return ['apitoken']
    }
    case 'datacenter': {
      return ['pat', 'oauth']
    }
    case 'server': {
      return ['pat', 'basic']
    }
  }
}

/**
 * Walks the user through connecting to a Confluence site: ask for the URL, detect the
 * edition, offer only the sign-in methods it accepts, collect the secret, verify it by
 * opening a throwaway session, and on success store the credential and metadata.
 *
 * A cancelled prompt ends the flow quietly. A failure from the engine — above all a
 * refused credential — is shown verbatim, so the message the user sees is the engine's.
 * Returns the stored connection, or `undefined` if cancelled or unverified.
 */
export async function setUpConnection (deps: { engine: EngineRequester; store: ConnectionStore; ui: ConnectionUI }): Promise<ConnectionMeta | undefined> {
  const { engine, store, ui } = deps

  const baseUrl = await ui.promptBaseUrl()
  if (!baseUrl) {
    return undefined
  }

  let detected: EditionDetectResult
  try {
    detected = await engine.request<EditionDetectResult>(EDITION_DETECT_METHOD, { baseUrl })
  } catch (error) {
    await ui.showError(messageOf(error))

    return undefined
  }

  const method = await ui.pickAuthMethod(authMethodsFor(detected.edition))
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
  // An interactive OAuth sign-in returns tokens: store them (and the client id) so later
  // sessions reuse the access token and the engine can refresh it, instead of signing in
  // through the browser every time.
  const stored: Credential = session.tokens
    ? { kind: 'oauth', accessToken: session.tokens.accessToken, refreshToken: session.tokens.refreshToken, clientId: credential.clientId }
    : credential
  await store.add(meta, stored)
  await ui.showConnected(meta)

  return meta
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
