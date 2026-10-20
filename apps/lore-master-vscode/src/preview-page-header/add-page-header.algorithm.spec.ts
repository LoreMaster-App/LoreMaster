import { addPageHeader } from './add-page-header.algorithm'
import type { HeaderMarkdownIt, HeaderState, HeaderToken, PageHeader } from './page-header.contract'

class FakeToken implements HeaderToken {
  content = ''
  block = false
  constructor (readonly type: string) {}
}

const noRule = (): void => {}

function render (headers: PageHeader[] | undefined, currentDocument: { fsPath: string } | undefined): HeaderToken[] {
  let rule: (state: HeaderState) => void = noRule
  const md: HeaderMarkdownIt = { core: { ruler: { push: (_name, made) => { rule = made } } } }
  addPageHeader(md, { headersOf: () => headers })
  const state: HeaderState = { tokens: [new FakeToken('heading_open')], env: { currentDocument }, Token: FakeToken }
  rule(state)

  return state.tokens
}

describe('addPageHeader', () => {
  it('puts the strip first', () => {
    const tokens = render([{ storage: 'Docs', pageTitle: 'Team: Guide', status: 'synced' }], { fsPath: '/repo/a.md' })

    expect(tokens).toHaveLength(2)
    expect((tokens[0] as FakeToken).type).toBe('html_block')
    expect(tokens[0].content).toContain('Team: Guide')
  })

  it('leaves the page as it was when there is nothing to say', () => {
    expect(render(undefined, { fsPath: '/repo/a.md' })).toHaveLength(1)
    expect(render([], { fsPath: '/repo/a.md' })).toHaveLength(1)
  })

  it('leaves a render with no document alone', () => {
    expect(render([{ storage: 'Docs', pageTitle: 'T' }], undefined)).toHaveLength(1)
  })
})
