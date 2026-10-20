import type { TreeStatus } from '../engine-protocol'

/** What the preview says about the open file for one storage that syncs it. */
export interface PageHeader {
  /** What the storage is called in the settings. */
  storage:   string
  /** The title the page gets on the platform: the file's title with the storage's prefix. */
  pageTitle: string
  /** What the file alone says about the page; absent for a storage that does not track pages in the files. */
  status?:   TreeStatus
}

/** What the preview asks for: the headers of a file, or nothing while they are still being fetched or there are none. */
export interface PageHeaderLookup {
  headersOf (documentPath: string): PageHeader[] | undefined
}

/** The one thing the lookup needs from the engine client: to make a request. */
export interface HeaderEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** The part of a markdown-it token the header rule creates. */
export interface HeaderToken {
  content: string
  block:   boolean
}

/** The part of markdown-it's core state the header rule reads and changes. */
export interface HeaderState {
  tokens: HeaderToken[]
  env:    { currentDocument?: { fsPath: string } }
  Token:  new (type: string, tag: string, nesting: number) => HeaderToken
}

/** The part of the markdown-it instance VS Code hands to an extension that the header rule needs. */
export interface HeaderMarkdownIt {
  core: { ruler: { push (name: string, rule: (state: HeaderState) => void): void } }
}
