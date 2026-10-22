/** The part of a markdown-it token the link rule reads and marks. */
export interface LinkToken {
  attrs: [string, string][] | null
  attrGet (name: string): string | null
  attrJoin (name: string, value: string): void
  attrSet (name: string, value: string): void
}

/** What a render knows: VS Code sets the open file when it renders, not when it parses. */
export interface LinkEnv {
  currentDocument?: { fsPath: string }
}

/** The part of markdown-it's renderer a rule of one token type is given. */
export interface LinkRenderer {
  renderToken (tokens: LinkToken[], index: number, options: unknown): string
}

/** How a token of one type is written as HTML. */
export type LinkRenderRule = (tokens: LinkToken[], index: number, options: unknown, env: LinkEnv, self: LinkRenderer) => string

/** The part of the markdown-it instance VS Code hands to an extension that the link rule needs. */
export interface LinkMarkdownIt {
  renderer: { rules: Record<string, LinkRenderRule | undefined> }
}

/** What the rule asks of the machine: whether a file is there, and whether a storage syncs it. */
export interface LinkTargets {
  exists (path: string): boolean
  /** Whether a storage syncs the file; undefined while that is not known yet. */
  isSynced (path: string): boolean | undefined
}

/** What is wrong with a link, if anything. */
export type LinkProblem = 'broken' | 'unsynced'
