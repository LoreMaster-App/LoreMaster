import { pageHeaderMarkup } from './page-header-markup.mapper'
import type { HeaderMarkdownIt, PageHeaderLookup } from './page-header.contract'

/**
 * Puts a strip at the top of the Markdown preview saying, for each storage that syncs the open file, the title its page will
 * get and where the page stands. A file no storage syncs, or one whose answer has not arrived yet, is left as it was.
 */
export function addPageHeader<Md extends HeaderMarkdownIt> (md: Md, lookup: PageHeaderLookup): Md {
  md.core.ruler.push('loremaster_page_header', state => {
    const path = state.env.currentDocument?.fsPath
    const headers = path === undefined ? undefined : lookup.headersOf(path)
    if (headers === undefined || headers.length === 0) { return }
    const token = new state.Token('html_block', '', 0)
    token.block = true
    token.content = pageHeaderMarkup(headers)
    state.tokens.unshift(token)
  })

  return md
}
