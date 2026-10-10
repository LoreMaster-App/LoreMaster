import { flatten, repoTree } from './arrange-entries.algorithm'

const item = (path: string): { path: string } => ({ path })
const pathOf = (each: { path: string }): string => each.path

describe('repoTree', () => {
  it('places files in their folders, folders before files, in name order', () => {
    const tree = repoTree(['README.md', 'docs/b.md', 'docs/adr/one.md', 'docs/a.md', 'apps/x/README.md'].map(each => item(each)), pathOf)

    expect(tree.files.map(each => pathOf(each))).toEqual(['README.md'])
    expect(tree.folders.map(folder => folder.name)).toEqual(['apps', 'docs'])
    const docs = tree.folders[1]
    expect(docs.path).toBe('docs')
    expect(docs.folders.map(folder => folder.path)).toEqual(['docs/adr'])
    expect(docs.files.map(each => pathOf(each))).toEqual(['docs/a.md', 'docs/b.md'])
  })

  it('has no folder for a folder with no file', () => {
    expect(repoTree([], pathOf).folders).toEqual([])
  })
})

describe('flatten', () => {
  it('orders by path, numbers read as numbers', () => {
    expect(flatten(['docs/10.md', 'docs/2.md', 'a.md'].map(each => item(each)), pathOf).map(each => pathOf(each))).toEqual(['a.md', 'docs/2.md', 'docs/10.md'])
  })
})
