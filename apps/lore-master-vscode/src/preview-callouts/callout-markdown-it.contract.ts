/** The part of a markdown-it token the callout rule reads and marks. */
export interface CalloutToken {
  type:     string
  content:  string
  children: CalloutToken[] | null
  attrJoin (name: string, value: string): void
  attrSet (name: string, value: string): void
}

/** The part of the markdown-it instance VS Code hands to an extension that the callout rule needs. */
export interface CalloutMarkdownIt {
  core: { ruler: { push (name: string, rule: (state: { tokens: CalloutToken[] }) => void): void } }
}

/** What VS Code's Markdown extension calls on an extension that contributes `markdown.markdownItPlugins`. */
export interface CalloutExtension {
  extendMarkdownIt<Md extends CalloutMarkdownIt> (md: Md): Md
}
