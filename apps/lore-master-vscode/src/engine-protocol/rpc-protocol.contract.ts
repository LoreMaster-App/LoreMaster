// The TypeScript mirror of the engine's JSON-RPC contract (#57). The source of truth is
// the Go package apps/lore-master-engine/rpcprotocol; rpc-protocol.contract.spec.ts fails
// if a method here drifts from there. Every slice that speaks to the engine imports these
// names rather than re-declaring them, so the wire shape lives in one place.

// ---- methods -------------------------------------------------------------------------

export const PING_METHOD = 'ping'
export const EDITION_DETECT_METHOD = 'edition/detect'
export const SESSION_OPEN_METHOD = 'session/open'
export const SESSION_CLOSE_METHOD = 'session/close'
export const SPACE_LIST_METHOD = 'space/list'
export const PAGE_CHILDREN_METHOD = 'page/children'
export const PAGE_SEARCH_METHOD = 'page/search'
export const SYNC_PLAN_METHOD = 'sync/plan'
export const SYNC_EXECUTE_METHOD = 'sync/execute'
export const SETTINGS_READ_METHOD = 'settings/read'
export const SETTINGS_SAVE_METHOD = 'settings/save'
export const PAGES_PUBLISH_METHOD = 'pages/publish'
export const HOST_PROGRESS_METHOD = 'host/progress'
export const HOST_RENDER_DIAGRAM_METHOD = 'host/renderDiagram'
export const CANCEL_REQUEST_METHOD = '$/cancelRequest'

// ---- shared vocabulary ---------------------------------------------------------------

/** A Confluence edition. */
export type Edition = 'cloud' | 'datacenter' | 'server'

/** How a user signs in; `kind` decides which fields are read: `apitoken` (Cloud) reads
 *  email + token, `pat` (Data Center/Server) reads token, `basic` reads user + password. */
export type CredentialKind = 'apitoken' | 'pat' | 'basic'

export interface Credential {
  kind:      CredentialKind
  email?:    string
  token?:    string
  user?:     string
  password?: string
}

// ---- ping ----------------------------------------------------------------------------

export interface PingResult {
  pong:    string
  version: string
}

// ---- edition/detect ------------------------------------------------------------------

export interface EditionDetectParams {
  baseUrl: string
}

export interface EditionDetectResult {
  baseUrl:  string
  edition:  Edition
  version?: string
}

// ---- session/open, session/close -----------------------------------------------------

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

export interface SessionCloseParams {
  sessionId: string
}

// ---- space/list, page/children, page/search ------------------------------------------

export interface Space {
  id:          string
  key:         string
  name:        string
  homepageId?: string
}

export interface SpaceListParams {
  sessionId: string
}

export interface SpaceListResult {
  spaces: Space[]
}

export interface Page {
  id:        string
  title:     string
  parentId?: string
  url?:      string
}

export interface PagesResult {
  pages: Page[]
}

export interface PageChildrenParams {
  sessionId: string
  pageId:    string
}

export interface PageSearchParams {
  sessionId: string
  spaceKey:  string
  query:     string
  limit?:    number
}

// ---- settings/read, settings/save ----------------------------------------------------

export interface Content {
  type:      string
  roots:     string[]
  excludes?: string[]
  template:  string
}

export interface Output {
  platform:       string
  baseUrl:        string
  space:          string
  parentPageId:   string
  titlePrefix:    string
  direction:      string
  content:        Content[]
  mermaidMode:    string
  titleCollision: string
  linkMode:       string
  /** github-pages: the repo ("owner/name" or a URL; empty = the workspace's own origin)
   *  and the branch to publish to (empty = gh-pages). */
  repo?:          string
  branch?:        string
}

export interface Settings {
  version:         number
  /** Leave out Markdown the workspace's .gitignore files ignore; absent means true. */
  skipGitignored?: boolean
  /** Gitignore-syntax patterns every output leaves out of the scan. */
  ignore?:         string[]
  outputs:         Output[]
}

export interface SettingsReadParams {
  workspaceRoot: string
}

export interface SettingsReadResult {
  exists:    boolean
  firstSync: boolean
  settings:  Settings
}

export interface SettingsSaveParams {
  workspaceRoot: string
  settings:      Settings
}

// ---- sync/plan, sync/execute ---------------------------------------------------------

export interface SyncPlanParams {
  sessionId:     string
  workspaceRoot: string
  output:        number
  scope?:        string[]
}

export interface PlanAction {
  kind:        string
  path?:       string
  title:       string
  pageId?:     string
  url?:        string
  parentPath?: string
  changes?:    string[]
  reason?:     string
}

export interface SyncPlanResult {
  planId:    string
  actions:   PlanAction[]
  counts:    Record<string, number>
  warnings?: string[]
  errors?:   string[]
}

export interface SyncExecuteParams {
  planId: string
  force?: boolean
  prune?: boolean
}

export interface PageOutcome {
  path?:    string
  title:    string
  planned:  string
  outcome:  string
  pageId?:  string
  version?: number
  url?:     string
  error?:   string
}

export interface SyncExecuteResult {
  pages:      PageOutcome[]
  rewritten?: string[]
  warnings?:  string[]
}

// ---- pages/publish -------------------------------------------------------------------

export interface PagesPublishParams {
  workspaceRoot: string
  output:        number
}

export interface PagesPublishResult {
  branch?:   string
  remote?:   string
  commit?:   string
  changed:   boolean
  files:     number
  url?:      string
  warnings?: string[]
  errors?:   string[]
}

// ---- notifications -------------------------------------------------------------------

export interface ProgressParams {
  planId:  string
  message: string
  done:    number
  total:   number
}

export interface RenderDiagramParams {
  language: string
  source:   string
}

export interface RenderDiagramResult {
  svg: string
}

export interface CancelParams {
  id: number | string
}
