import { addActionBar } from './add-action-bar.algorithm'
import type { BarMarkdownIt, BarRenderRule, BarState, BarToken, PageStatus } from './action-bar.contract'

class FakeToken implements BarToken {
  content = ''
  block = false
  constructor (readonly type: string) {}
}

const noRule = (): void => {}

/** Parses a page, then renders it for `document`, the way VS Code does: the open file is known only to the render. */
function render (status: PageStatus, document: { fsPath: string, toString (): string } | undefined, enabled = true): { tokens: BarToken[], html: string } {
  let rule: (state: BarState) => void = noRule
  const rules: Record<string, BarRenderRule | undefined> = {}
  const md: BarMarkdownIt = { core: { ruler: { push: (_name, made) => { rule = made } } }, renderer: { rules } }
  addActionBar(md, { statusOf: () => status }, () => enabled)
  const state: BarState = { tokens: [new FakeToken('heading_open')], Token: FakeToken }
  rule(state)
  const bar = rules[(state.tokens[0] as FakeToken).type]

  return { tokens: state.tokens, html: bar?.(state.tokens, 0, {}, { currentDocument: document }) ?? '' }
}

const file = { fsPath: '/repo/a.md', toString: () => 'file:///repo/a.md' }

describe('addActionBar', () => {
  it('puts the bar first, whatever the status, and writes it when the page is rendered', () => {
    const { tokens, html } = render({ state: 'checking' }, file)

    expect(tokens).toHaveLength(2)
    expect(html).toContain('loremaster-action-bar')
  })

  it('writes nothing when the setting turns the bar off', () => {
    expect(render({ state: 'checking' }, file, false).html).toBe('')
  })

  it('writes nothing in a render with no document', () => {
    expect(render({ state: 'checking' }, undefined).html).toBe('')
  })
})
