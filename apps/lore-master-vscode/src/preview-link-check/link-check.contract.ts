/** The part of a markdown-it token the link rule reads and marks. */
export interface LinkToken {
  type:     string
  children: LinkToken[] | null
  attrGet (name: string): string | null
  attrJoin (name: string, value: string): void
  attrSet (name: string, value: string): void
}

/** The part of markdown-it's core state the link rule reads. */
export interface LinkState {
  tokens: LinkToken[]
  env:    { currentDocument?: { fsPath: string } }
}

/** The part of the markdown-it instance VS Code hands to an extension that the link rule needs. */
export interface LinkMarkdownIt {
  core: { ruler: { push (name: string, rule: (state: LinkState) => void): void } }
}

/** What the rule asks of the machine: whether a file is there, and whether a storage syncs it. */
export interface LinkTargets {
  exists (path: string): boolean
  /** Whether a storage syncs the file; undefined while that is not known yet. */
  isSynced (path: string): boolean | undefined
}

/** What is wrong with a link, if anything. */
export type LinkProblem = 'broken' | 'unsynced'
