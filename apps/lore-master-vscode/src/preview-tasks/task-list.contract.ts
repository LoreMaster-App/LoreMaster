/** The part of a markdown-it token the task list rule reads and changes. */
export interface TaskToken {
  type:     string
  content:  string
  /** The lines the token covers, start first (0-based), when it covers any. */
  map:      [number, number] | null
  children: TaskToken[] | null
  meta:     unknown
  attrJoin (name: string, value: string): void
}

/** What a render knows: VS Code sets the open file when it renders, not when it parses. */
export interface TaskEnv {
  currentDocument?: { toString (): string }
}

/** How a token of one type is written as HTML. */
export type TaskRenderRule = (tokens: TaskToken[], index: number, options: unknown, env: TaskEnv) => string

/** The part of the markdown-it instance VS Code hands to an extension that the task list rule needs. */
export interface TaskMarkdownIt {
  core:     { ruler: { push (name: string, rule: (state: TaskState) => void): void } }
  renderer: { rules: Record<string, TaskRenderRule | undefined> }
}

/** The part of markdown-it's core state the task list rule changes. */
export interface TaskState {
  tokens: TaskToken[]
  Token:  new (type: string, tag: string, nesting: number) => TaskToken
}

/** What a task token remembers: whether the item is done and the line of the file it is written on. */
export interface TaskMeta {
  checked: boolean
  line:    number
}
