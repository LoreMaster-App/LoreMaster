import type { FootnoteMarkdownIt, FootnoteState, FootnoteToken } from './footnote-markdown-it.contract'

const DEFINITION = /^\[\^([^\]\s]+)\]:/
const REFERENCE = /\[\^([^\]\s]+)\]/g

const slug = (id: string): string => id.replaceAll(/[^\w-]/g, '-')

/** Takes the paragraphs that are footnote definitions out of the token list and returns them by id. */
function takeDefinitions (tokens: FootnoteToken[]): Map<string, string> {
  const definitions = new Map<string, string>()
  for (let index = 0; index + 2 < tokens.length; index++) {
    const [open, inline, close] = [tokens[index], tokens[index + 1], tokens[index + 2]]
    if (open.type !== 'paragraph_open' || inline.type !== 'inline' || close.type !== 'paragraph_close') { continue }
    const match = DEFINITION.exec(inline.content)
    if (match === null) { continue }
    definitions.set(match[1], inline.content.slice(match[0].length).replaceAll('\n', ' ').trim())
    tokens.splice(index, 3)
    index--
  }

  return definitions
}

/** Splits a text token at its references to defined footnotes, numbering them in the order they first appear. */
function referencesIn (text: FootnoteToken, state: FootnoteState, definitions: Map<string, string>, numbers: Map<string, number>): FootnoteToken[] {
  const parts: FootnoteToken[] = []
  let last = 0
  for (const match of text.content.matchAll(REFERENCE)) {
    const id = match[1]
    if (!definitions.has(id)) { continue }
    if (!numbers.has(id)) { numbers.set(id, numbers.size + 1) }
    const before = new state.Token('text', '', 0)
    before.content = text.content.slice(last, match.index)
    const reference = new state.Token('html_inline', '', 0)
    reference.content = `<sup class="loremaster-footnote-ref"><a id="fnref-${slug(id)}" href="#fn-${slug(id)}">[${numbers.get(id)}]</a></sup>`
    parts.push(before, reference)
    last = match.index + match[0].length
  }
  const rest = new state.Token('text', '', 0)
  rest.content = text.content.slice(last)

  return last === 0 ? [text] : [...parts, rest]
}

/**
 * Teaches the Markdown preview footnotes, which VS Code does not draw: `text[^1]` becomes a numbered link and a line `[^1]: The note`
 * becomes an entry of a list at the end of the page, with a way back. A definition is one paragraph; a reference to a note that is not
 * defined is left as written.
 */
export function addFootnotes<Md extends FootnoteMarkdownIt> (md: Md): Md {
  md.core.ruler.push('loremaster_footnotes', state => {
    const definitions = takeDefinitions(state.tokens)
    if (definitions.size === 0) { return }
    const numbers = new Map<string, number>()
    for (const token of state.tokens) {
      if (token.type !== 'inline' || token.children === null) { continue }
      token.children = token.children.flatMap(child => child.type === 'text' ? referencesIn(child, state, definitions, numbers) : [child])
    }
    if (numbers.size === 0) { return }
    const items = [...numbers].map(([id]) =>
      `<li id="fn-${slug(id)}">${md.renderInline(definitions.get(id) ?? '', state.env)} <a class="loremaster-footnote-back" href="#fnref-${slug(id)}" title="Back to the text">↩</a></li>`)
    const list = new state.Token('html_block', '', 0)
    list.block = true
    list.content = `<hr class="loremaster-footnotes-rule"><ol class="loremaster-footnotes">${items.join('')}</ol>\n`
    state.tokens.push(list)
  })

  return md
}
