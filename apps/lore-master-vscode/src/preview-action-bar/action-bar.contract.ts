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

/** What a render knows: VS Code sets the open file when it renders, not when it parses. */
export interface BarEnv {
  currentDocument?: { fsPath: string, toString (): string }
}

/** The part of markdown-it's core state the bar rule changes. */
export interface BarState {
  tokens: BarToken[]
  Token:  new (type: string, tag: string, nesting: number) => BarToken
}

/** How a token of one type is written as HTML. */
export type BarRenderRule = (tokens: BarToken[], index: number, options: unknown, env: BarEnv) => string

/** The part of the markdown-it instance VS Code hands to an extension that the bar needs. */
export interface BarMarkdownIt {
  core:     { ruler: { push (name: string, rule: (state: BarState) => void): void } }
  renderer: { rules: Record<string, BarRenderRule | undefined> }
}
