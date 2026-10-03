import type { Credential } from '../secret-storage'

// The slice of the engine's connect contract the setup flow uses. Mirrors the Go source
// in apps/lore-master-engine/rpcprotocol. The full TypeScript mirror is #57.

/** A Confluence edition. */
export type Edition = 'cloud' | 'datacenter' | 'server'

/** `edition/detect`: probe a site's edition without a credential. */
export const EDITION_DETECT_METHOD = 'edition/detect'

export interface EditionDetectResult {
  baseUrl:  string
  edition:  Edition
  version?: string
}

/** `session/open`: connect and verify, keeping the credential in the engine's memory. */
export const SESSION_OPEN_METHOD = 'session/open'

export interface SessionOpenParams {
  baseUrl:    string
  edition?:   Edition
  credential: Credential
}

export interface SessionUser {
  displayName: string
  accountId?:  string
  username?:   string
}

export interface SessionOpenResult {
  sessionId: string
  baseUrl:   string
  edition:   Edition
  version?:  string
  user:      SessionUser
}

/** `session/close`: forget a session and its credential. */
export const SESSION_CLOSE_METHOD = 'session/close'

export interface SessionCloseParams {
  sessionId: string
}
