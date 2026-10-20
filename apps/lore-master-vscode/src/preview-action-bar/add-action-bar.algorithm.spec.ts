import { addActionBar } from './add-action-bar.algorithm'
import type { BarMarkdownIt, BarState, BarToken, PageStatus } from './action-bar.contract'

class FakeToken implements BarToken {
  content = ''
  block = false
  constructor (readonly type: string) {}
}

const noRule = (): void => {}

function render (status: PageStatus, document: { fsPath: string, toString (): string } | undefined, enabled = true): BarToken[] {
  let rule: (state: BarState) => void = noRule
  const md: BarMarkdownIt = { core: { ruler: { push: (_name, made) => { rule = made } } } }
  addActionBar(md, { statusOf: () => status }, () => enabled)
  const state: BarState = { tokens: [new FakeToken('heading_open')], env: { currentDocument: document }, Token: FakeToken }
  rule(state)

  return state.tokens
}

const file = { fsPath: '/repo/a.md', toString: () => 'file:///repo/a.md' }

describe('addActionBar', () => {
  it('puts the bar first, whatever the status', () => {
    const tokens = render({ state: 'checking' }, file)

    expect(tokens).toHaveLength(2)
    expect((tokens[0] as FakeToken).type).toBe('html_block')
    expect(tokens[0].content).toContain('loremaster-action-bar')
  })

  it('leaves the page as it was when the setting turns the bar off', () => {
    expect(render({ state: 'checking' }, file, false)).toHaveLength(1)
  })

  it('leaves a render with no document alone', () => {
    expect(render({ state: 'checking' }, undefined)).toHaveLength(1)
  })
})
