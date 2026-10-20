import { addCalloutPanels } from './callout-panel.algorithm'
import type { CalloutMarkdownIt, CalloutToken } from './callout-markdown-it.contract'

function token (type: string, content = '', children: CalloutToken[] | null = null): CalloutToken & { attrs: Record<string, string> } {
  const attrs: Record<string, string> = {}

  return {
    type,
    content,
    children,
    attrs,
    attrJoin: (name, value) => { attrs[name] = [attrs[name], value].filter(Boolean).join(' ') },
    attrSet:  (name, value) => { attrs[name] = value },
  }
}

const noRule = (): void => {}

function run (tokens: CalloutToken[]): void {
  let rule: (state: { tokens: CalloutToken[] }) => void = noRule
  const md: CalloutMarkdownIt = { core: { ruler: { push: (_name, made) => { rule = made } } } }
  addCalloutPanels(md)
  rule({ tokens })
}

const quoteOf = (content: string, children: CalloutToken[]) => {
  const quote = token('blockquote_open')

  return { quote, inline: token('inline', content, children), tokens: [quote, token('paragraph_open')] }
}

describe('addCalloutPanels', () => {
  it('marks a [!WARNING] blockquote as a panel and removes the marker line', () => {
    const children = [token('text', '[!WARNING]'), token('softbreak'), token('text', 'Mind the gap')]
    const { quote, inline, tokens } = quoteOf('[!WARNING]\nMind the gap', children)

    run([...tokens, inline])

    expect(quote.attrs).toEqual({ 'class': 'loremaster-callout loremaster-callout-warning', 'data-callout': 'Warning' })
    expect(inline.content).toBe('Mind the gap')
    expect(inline.children?.map(child => child.content)).toEqual(['Mind the gap'])
  })

  it('accepts the marker in any case', () => {
    const { quote, inline, tokens } = quoteOf('[!note]', [token('text', '[!note]')])

    run([...tokens, inline])

    expect(quote.attrs['data-callout']).toBe('Note')
  })

  it('leaves an ordinary blockquote alone', () => {
    const { quote, inline, tokens } = quoteOf('Just a quote', [token('text', 'Just a quote')])

    run([...tokens, inline])

    expect(quote.attrs).toEqual({})
    expect(inline.content).toBe('Just a quote')
  })

  it('leaves an unknown kind alone', () => {
    const { quote, inline, tokens } = quoteOf('[!DANGER]\nx', [token('text', '[!DANGER]')])

    run([...tokens, inline])

    expect(quote.attrs).toEqual({})
  })
})
