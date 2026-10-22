import { addFootnotes } from './add-footnotes.algorithm'
import type { FootnoteMarkdownIt, FootnoteState, FootnoteToken } from './footnote-markdown-it.contract'

class FakeToken implements FootnoteToken {
  content = ''
  block = false
  children: FootnoteToken[] | null = null
  constructor (readonly type: string) {}
}

const make = (type: string, content = ''): FootnoteToken => Object.assign(new FakeToken(type), { content })

const paragraph = (content: string, children?: FootnoteToken[]): FootnoteToken[] => {
  const inline = make('inline', content)
  inline.children = children ?? [make('text', content)]

  return [make('paragraph_open'), inline, make('paragraph_close')]
}

const noRule = (): void => {}

function run (tokens: FootnoteToken[]): FootnoteToken[] {
  let rule: (state: FootnoteState) => void = noRule
  const md: FootnoteMarkdownIt = {
    core:         { ruler: { push: (_name, made) => { rule = made } } },
    renderInline: source => `<i>${source}</i>`,
  }
  addFootnotes(md)
  rule({ tokens, env: {}, Token: FakeToken })

  return tokens
}

const html = (tokens: FootnoteToken[]): string => tokens.flatMap(token => token.children ?? [token]).map(token => token.type === 'text' ? token.content : token.content).join('')

describe('addFootnotes', () => {
  it('numbers the references, removes the definitions and lists the notes at the end', () => {
    const tokens = run([...paragraph('One[^b] and two[^a].'), ...paragraph('[^a]: Note A'), ...paragraph('[^b]: Note B')])

    expect(tokens.map(token => token.type)).toEqual(['paragraph_open', 'inline', 'paragraph_close', 'html_block'])
    const text = html(tokens)
    expect(text).toContain('<a id="fnref-b" href="#fn-b">[1]</a>')
    expect(text).toContain('<a id="fnref-a" href="#fn-a">[2]</a>')
    expect(tokens[3].content.indexOf('id="fn-b"')).toBeLessThan(tokens[3].content.indexOf('id="fn-a"'))
    expect(tokens[3].content).toContain('<i>Note B</i>')
  })

  it('reads definitions on consecutive lines as separate notes, and a following line as a continuation', () => {
    const tokens = run([...paragraph('One[^a] and two[^b].'), ...paragraph('[^a]: Note A\n[^b]: Note B\nand more')])

    expect(tokens.map(token => token.type)).toEqual(['paragraph_open', 'inline', 'paragraph_close', 'html_block'])
    expect(tokens[3].content).toContain('<i>Note A</i>')
    expect(tokens[3].content).toContain('<i>Note B and more</i>')
  })

  it('leaves a page without definitions as it was', () => {
    const tokens = run(paragraph('A reference[^1] to nothing.'))

    expect(tokens).toHaveLength(3)
    expect(html(tokens)).toBe('A reference[^1] to nothing.')
  })

  it('leaves a reference to an undefined note as written', () => {
    const tokens = run([...paragraph('Defined[^1] and not[^2].'), ...paragraph('[^1]: Yes')])

    expect(html(tokens)).toContain('and not[^2].')
    expect(tokens.at(-1)?.content).not.toContain('fn-2')
  })

  it('adds no list for a note that is never referenced', () => {
    const tokens = run([...paragraph('Nothing here.'), ...paragraph('[^1]: Orphan')])

    expect(tokens.map(token => token.type)).toEqual(['paragraph_open', 'inline', 'paragraph_close'])
  })
})
