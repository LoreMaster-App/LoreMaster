import { join, resolve } from 'node:path'
import { addLinkChecks } from './add-link-checks.algorithm'
import type { LinkMarkdownIt, LinkRenderer, LinkRenderRule, LinkToken } from './link-check.contract'

const DOC = join('repo', 'docs', 'a.md')
const B = resolve('repo', 'docs', 'b.md')

/** A link token as markdown-it makes it: a list of name and value pairs. */
class FakeLink implements LinkToken {
  constructor (public attrs: [string, string][] | null) {}

  attrGet (name: string): string | null {
    return this.attrs?.find(pair => pair[0] === name)?.[1] ?? null
  }

  attrSet (name: string, value: string): void {
    const found = this.attrs?.find(pair => pair[0] === name)
    if (found) {
      found[1] = value
    } else {
      this.attrs = [...this.attrs ?? [], [name, value]]
    }
  }

  attrJoin (name: string, value: string): void {
    this.attrSet(name, [this.attrGet(name), value].filter(Boolean).join(' '))
  }
}

interface Targets { exists: boolean, synced: Record<string, boolean | undefined> }

/** Renders a link the way VS Code does: the page is parsed without knowing the file, and the file is given to the render. */
function drawn (href: string, targets: Targets, document: string | null = DOC): { attrs: Record<string, string>, token: FakeLink } {
  const rules: Record<string, LinkRenderRule | undefined> = {}
  const md: LinkMarkdownIt = { renderer: { rules } }
  addLinkChecks(md, { exists: () => targets.exists, isSynced: path => targets.synced[path] })
  let attrs: Record<string, string> = {}
  const renderer: LinkRenderer = {
    renderToken: (tokens, index) => {
      attrs = Object.fromEntries(tokens[index].attrs ?? [])

      return ''
    },
  }
  const token = new FakeLink([['href', href]])
  const env = { currentDocument: document === null ? undefined : { fsPath: document } }
  rules.link_open?.([token], 0, {}, env, renderer)

  return { attrs, token }
}

const check = (href: string, targets: Targets): Record<string, string> => drawn(href, targets).attrs

const BOTH = { [DOC]: true, [B]: true }

describe('addLinkChecks', () => {
  it('marks a link to a missing file as broken', () => {
    const attrs = check('./gone.md', { exists: false, synced: BOTH })

    expect(attrs.class).toBe('loremaster-link-broken')
    expect(attrs.title).toContain('gone.md')
  })

  it('marks a link from a synced file to one no storage syncs', () => {
    const attrs = check('./b.md', { exists: true, synced: { [DOC]: true, [B]: false } })

    expect(attrs.class).toBe('loremaster-link-unsynced')
  })

  it('leaves a resolving link alone, whatever follows the file name', () => {
    expect(check('./b.md#part', { exists: true, synced: BOTH }).class).toBeUndefined()
    expect(check('./b.md?x=1', { exists: true, synced: BOTH }).class).toBeUndefined()
  })

  it('does not call a link unsynced while it is not known, or when the page itself is not synced', () => {
    expect(check('./b.md', { exists: true, synced: { [DOC]: true, [B]: undefined } }).class).toBeUndefined()
    expect(check('./b.md', { exists: true, synced: { [DOC]: false, [B]: false } }).class).toBeUndefined()
  })

  it('draws the mark without changing the parsed page, which VS Code renders again', () => {
    const { token, attrs } = drawn('./gone.md', { exists: false, synced: BOTH })

    expect(attrs.class).toBe('loremaster-link-broken')
    expect(token.attrs).toEqual([['href', './gone.md']])
  })

  it('marks nothing in a render that does not say which file is open', () => {
    expect(drawn('./gone.md', { exists: false, synced: BOTH }, null).attrs).toEqual({ href: './gone.md' })
  })

  it('ignores addresses, in-page links and other files', () => {
    for (const href of ['https://example.com/x.md', '#top', './image.png', 'mailto:a@b.c', '']) {
      expect(check(href, { exists: false, synced: {} }).class).toBeUndefined()
    }
  })
})
