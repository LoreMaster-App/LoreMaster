import { join, resolve } from 'node:path'
import { addLinkChecks } from './add-link-checks.algorithm'
import type { LinkMarkdownIt, LinkState, LinkToken } from './link-check.contract'

const DOC = join('repo', 'docs', 'a.md')
const B = resolve('repo', 'docs', 'b.md')

function link (href: string): LinkToken & { attrs: Record<string, string> } {
  const attrs: Record<string, string> = { href }

  return {
    type:     'link_open',
    children: null,
    attrs,
    attrGet:  name => attrs[name] ?? null,
    attrJoin: (name, value) => { attrs[name] = [attrs[name], value].filter(Boolean).join(' ') },
    attrSet:  (name, value) => { attrs[name] = value },
  }
}

const noRule = (): void => {}

function check (href: string, targets: { exists: boolean, synced: Record<string, boolean | undefined> }): Record<string, string> {
  let rule: (state: LinkState) => void = noRule
  const md: LinkMarkdownIt = { core: { ruler: { push: (_name, made) => { rule = made } } } }
  addLinkChecks(md, { exists: () => targets.exists, isSynced: path => targets.synced[path] })
  const made = link(href)
  rule({ tokens: [{ type: 'inline', children: [made], attrGet: () => null, attrJoin: noRule, attrSet: noRule }], env: { currentDocument: { fsPath: DOC } } })

  return made.attrs
}

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

  it('ignores addresses, in-page links and other files', () => {
    for (const href of ['https://example.com/x.md', '#top', './image.png', 'mailto:a@b.c', '']) {
      expect(check(href, { exists: false, synced: {} }).class).toBeUndefined()
    }
  })
})
