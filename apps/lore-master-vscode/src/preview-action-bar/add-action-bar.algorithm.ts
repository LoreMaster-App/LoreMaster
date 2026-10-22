import { actionBarMarkup } from './action-bar-markup.mapper'
import type { BarMarkdownIt, PageStatusLookup } from './action-bar.contract'

const BAR_TOKEN = 'loremaster_action_bar'

/**
 * Puts the action bar at the top of the Markdown preview: where the open file stands with the storages that sync it, an Export
 * button and the switches for LoreMaster's additions. VS Code parses a file once and renders it again and again, and only tells
 * the renderer which file is open, so the page gets a token of its own and the bar is written when the page is rendered.
 * `enabled` is read at every render, so the setting takes effect at once.
 */
export function addActionBar<Md extends BarMarkdownIt> (md: Md, lookup: PageStatusLookup, enabled: () => boolean): Md {
  md.core.ruler.push(BAR_TOKEN, state => {
    const token = new state.Token(BAR_TOKEN, '', 0)
    token.block = true
    state.tokens.unshift(token)
  })
  md.renderer.rules[BAR_TOKEN] = (_tokens, _index, _options, env) => {
    const document = env.currentDocument
    if (document === undefined || !enabled()) { return '' }

    return actionBarMarkup(document.toString(), lookup.statusOf(document.fsPath))
  }

  return md
}
