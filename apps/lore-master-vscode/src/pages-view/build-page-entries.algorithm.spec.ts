import type { TreeNode } from '../engine-protocol'
import { buildPageEntries } from './build-page-entries.algorithm'
import type { PageEntry } from './page-tree.contract'

function node (path: string, title: string, parent?: string, status?: TreeNode['status']): TreeNode {
  return { path, title, pageTitle: `P: ${title}`, parent, rule: 'selected-parent', depth: 0, status }
}

const paths = (entries: PageEntry[]): unknown[] => entries.map(entry => entry.children.length > 0 ? [entry.node.path, paths(entry.children)] : entry.node.path)

describe('buildPageEntries', () => {
  const nodes = [
    node('README.md', 'Home', undefined, 'synced'),
    node('readme.setup.md', 'Setup', 'README.md', 'new'),
    node('10-late.md', 'Late'),
    node('2-early.md', 'Early'),
    node('docs/guide.md', 'Guide', 'README.md', 'local-changes'),
  ]

  it('nests children under their parent and keeps the engine order in title mode', () => {
    expect(paths(buildPageEntries(nodes, 'title'))).toEqual([
      ['README.md', ['readme.setup.md', 'docs/guide.md']], '10-late.md', '2-early.md',
    ])
  })

  it('sorts siblings by file name, reading numbers as numbers, in file name mode', () => {
    expect(paths(buildPageEntries(nodes, 'fileName'))).toEqual([
      '2-early.md', '10-late.md', ['README.md', ['docs/guide.md', 'readme.setup.md']],
    ])
  })

  it('puts a node whose parent is missing at the top', () => {
    const entries = buildPageEntries([node('a.md', 'A', 'gone.md')], 'title')

    expect(paths(entries)).toEqual(['a.md'])
  })

  it('carries the local status when there is no remote check', () => {
    const [home] = buildPageEntries(nodes, 'title')
    expect(home.status).toBe('synced')
    expect(home.remoteChecked).toBe(false)
  })

  it('combines the remote plan into the status and marks the page as checked', () => {
    const [home] = buildPageEntries(nodes, 'title', { actions: { 'README.md': 'conflict', 'docs/guide.md': 'conflict' }, orphans: [] })
    expect(home.status).toBe('remote-changes')
    expect(home.remoteChecked).toBe(true)
    expect(home.children.find(child => child.node.path === 'docs/guide.md')?.status).toBe('conflict')
    expect(home.children.find(child => child.node.path === 'readme.setup.md')?.remoteChecked).toBe(false)
  })

  it('leaves the status empty for an output that does not track pages', () => {
    expect(buildPageEntries([node('a.md', 'A')], 'title')[0].status).toBeUndefined()
  })
})
