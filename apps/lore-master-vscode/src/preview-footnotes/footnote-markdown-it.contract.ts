/** The part of a markdown-it token the footnote rule reads and changes. */
export interface FootnoteToken {
  type:     string
  content:  string
  block:    boolean
  children: FootnoteToken[] | null
}

/** The part of markdown-it's core state the footnote rule reads and changes. */
export interface FootnoteState {
  tokens: FootnoteToken[]
  env:    unknown
  Token:  new (type: string, tag: string, nesting: number) => FootnoteToken
}

/** The part of the markdown-it instance VS Code hands to an extension that the footnote rule needs. */
export interface FootnoteMarkdownIt {
  core: { ruler: { push (name: string, rule: (state: FootnoteState) => void): void } }
  renderInline (source: string, env?: unknown): string
}
