import type { CalloutMarkdownIt, CalloutToken } from './callout-markdown-it.contract'

/** The GitHub alert kinds and the title each panel carries. Confluence draws them as its info, note and warning macros. */
const TITLES: Record<string, string> = {
  note:      'Note',
  tip:       'Tip',
  important: 'Important',
  warning:   'Warning',
  caution:   'Caution',
}

const MARKER = /^\[!(note|tip|important|warning|caution)\][ \t]*(?:\n|$)/i

/** Drops the marker line from the paragraph's inline children so it is not drawn twice. */
function dropMarker (inline: CalloutToken): void {
  inline.content = inline.content.replace(MARKER, '')
  const children = inline.children ?? []
  children.shift()
  if (children[0]?.type === 'softbreak') { children.shift() }
}

/**
 * Teaches the Markdown preview GitHub alerts: a blockquote whose first line is `[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`,
 * `[!WARNING]` or `[!CAUTION]` becomes a panel. The blockquote gets a kind class and its title as `data-callout`; the preview
 * style draws the rest.
 */
export function addCalloutPanels<Md extends CalloutMarkdownIt> (md: Md): Md {
  md.core.ruler.push('loremaster_callouts', state => {
    const { tokens } = state
    for (let index = 0; index + 2 < tokens.length; index++) {
      const [quote, paragraph, inline] = [tokens[index], tokens[index + 1], tokens[index + 2]]
      if (quote.type !== 'blockquote_open' || paragraph.type !== 'paragraph_open' || inline.type !== 'inline') { continue }
      const kind = MARKER.exec(inline.content)?.[1].toLowerCase()
      if (kind === undefined) { continue }
      quote.attrJoin('class', 'loremaster-callout')
      quote.attrJoin('class', `loremaster-callout-${kind}`)
      quote.attrSet('data-callout', TITLES[kind])
      dropMarker(inline)
    }
  })

  return md
}
