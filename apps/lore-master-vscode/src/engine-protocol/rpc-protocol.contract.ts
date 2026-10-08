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
export const WORKSPACE_TREE_METHOD = 'workspace/tree'
export const GENERATORS_RUN_METHOD = 'generators/run'
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

/** One generator of .lore-master.yaml: what it reads and the folder it writes its pages to. */
export interface Generator {
  /** test-results (JUnit XML); other types are reserved. */
  type:   string
  /** Gitignore-style patterns selecting what to read; absent means the type's default. */
  input?: string[]
  /** Workspace-relative folder the pages are written to. */
  output: string
  title?: string
}

export interface Settings {
  version:         number
  /** Leave out Markdown the workspace's .gitignore files ignore; absent means true. */
  skipGitignored?: boolean
  /** Gitignore-syntax patterns every output leaves out of the scan. */
  ignore?:         string[]
  /** Write Markdown into the workspace from test reports and other artifacts. */
  generators?:     Generator[]
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

// ---- workspace/tree ------------------------------------------------------------------

export interface WorkspaceTreeParams {
  workspaceRoot: string
  /** Index of the output in the settings' outputs list. */
  output:        number
}

/** What the files alone say about a page; the remote half comes from a read-only sync/plan. */
export type TreeStatus = 'new' | 'synced' | 'local-changes'

export interface TreeNode {
  /** Workspace-relative file, '/'-separated. */
  path:      string
  /** The title from the file (H1, front-matter or annotation title, else the file name). */
  title:     string
  /** What the platform shows: the title with the output's prefix. */
  pageTitle: string
  /** The file this page nests under; absent means directly under the configured parent. */
  parent?:   string
  rule:      string
  depth:     number
  /** Absent for an output that does not track pages in the files (github-pages). */
  status?:   TreeStatus
  pageId?:   string
  warnings?: string[]
}

export interface WorkspaceTreeResult {
  /** Parents first. */
  nodes:     TreeNode[]
  warnings?: string[]
  problems?: string[]
}

// ---- generators/run ------------------------------------------------------------------

export interface GeneratorsRunParams {
  workspaceRoot: string
  /** Indexes into the settings' generators list; absent runs them all. */
  generators?:   number[]
}

export interface GeneratorRun {
  index:      number
  type:       string
  output:     string
  written?:   string[]
  unchanged?: string[]
  removed?:   string[]
  warnings?:  string[]
  /** Why the generator could not run at all. */
  error?:     string
}

export interface GeneratorsRunResult {
  runs: GeneratorRun[]
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
