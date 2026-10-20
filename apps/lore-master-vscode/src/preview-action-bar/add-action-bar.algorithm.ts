import { actionBarMarkup } from './action-bar-markup.mapper'
import type { BarMarkdownIt, PageStatusLookup } from './action-bar.contract'

/**
 * Puts the action bar at the top of the Markdown preview: where the open file stands with the storages that sync it, an Export
 * button and the switches for LoreMaster's additions. `enabled` is read at every render, so the setting takes effect at once.
 */
export function addActionBar<Md extends BarMarkdownIt> (md: Md, lookup: PageStatusLookup, enabled: () => boolean): Md {
  md.core.ruler.push('loremaster_action_bar', state => {
    const document = state.env.currentDocument
    if (document === undefined || !enabled()) { return }
    const token = new state.Token('html_block', '', 0)
    token.block = true
    token.content = actionBarMarkup(document.toString(), lookup.statusOf(document.fsPath))
    state.tokens.unshift(token)
  })

  return md
}
