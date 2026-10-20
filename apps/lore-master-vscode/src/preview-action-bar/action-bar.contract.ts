import type { TreeStatus } from '../engine-protocol'

/** What the bar says about the open file for one storage that syncs it. */
export interface PageHeader {
  /** What the storage is called in the settings. */
  storage:   string
  /** The title the page gets on the platform: the file's title with the storage's prefix. */
  pageTitle: string
  /** What the file alone says about the page; absent for a storage that does not track pages in the files. */
  status?:   TreeStatus
}

/** Where the open file stands with the storages, as far as the engine has said. */
export type PageStatus =
  | { state: 'checking' } |
  { state: 'untracked' } |
  { state: 'unavailable', reason: string } |
  { state: 'tracked', pages: PageHeader[] }

/** What the bar asks for: where a file stands. A file not yet known is `checking` and the answer is fetched. */
export interface PageStatusLookup {
  statusOf (documentPath: string): PageStatus
}

/** The one thing the lookup needs from the engine client: to make a request. */
export interface StatusEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** The part of a markdown-it token the bar rule creates. */
export interface BarToken {
  content: string
  block:   boolean
}

/** The part of markdown-it's core state the bar rule reads and changes. */
export interface BarState {
  tokens: BarToken[]
  env:    { currentDocument?: { fsPath: string, toString (): string } }
  Token:  new (type: string, tag: string, nesting: number) => BarToken
}

/** The part of the markdown-it instance VS Code hands to an extension that the bar rule needs. */
export interface BarMarkdownIt {
  core: { ruler: { push (name: string, rule: (state: BarState) => void): void } }
}
