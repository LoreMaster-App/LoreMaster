import type { Output } from '../engine-protocol'
import { freeOutputFolder, isFolderSynced, outputProblem, syncedRoots } from './generator-placement.algorithm'

const storage = (roots: string[], type = 'markdown'): Output => ({
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type, roots, template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
})

describe('freeOutputFolder', () => {
  it('uses the base when nothing writes there', () => {
    expect(freeOutputFolder('docs/tests', [])).toBe('docs/tests')
    expect(freeOutputFolder('docs/tests', ['docs/api'])).toBe('docs/tests')
  })

  it('numbers the folder when the base is taken, ignoring a trailing slash', () => {
    expect(freeOutputFolder('docs/tests', ['docs/tests'])).toBe('docs/tests-2')
    expect(freeOutputFolder('docs/tests', ['docs/tests/', 'docs/tests-2'])).toBe('docs/tests-3')
  })
})

describe('syncedRoots', () => {
  it('lists the roots of every Markdown entry of every storage, once', () => {
    expect(syncedRoots([storage(['docs', 'guides']), storage(['docs', 'adr'])])).toEqual(['docs', 'guides', 'adr'])
  })

  it('ignores entries that are not Markdown', () => {
    expect(syncedRoots([storage(['x'], 'test-results'), storage(['docs'])])).toEqual(['docs'])
  })
})

describe('isFolderSynced', () => {
  it('is true for a folder inside a root, the root itself, or any folder when the whole workspace is read', () => {
    expect(isFolderSynced('docs/tests', [storage(['docs'])])).toBe(true)
    expect(isFolderSynced('docs', [storage(['docs/'])])).toBe(true)
    expect(isFolderSynced('anywhere/at/all', [storage(['.'])])).toBe(true)
    expect(isFolderSynced('docs/tests', [storage(['./docs'])])).toBe(true)
  })

  it('is false for a folder outside every root, including one that only shares a prefix', () => {
    expect(isFolderSynced('reports', [storage(['docs'])])).toBe(false)
    expect(isFolderSynced('docs-extra/tests', [storage(['docs'])])).toBe(false)
    expect(isFolderSynced('docs/tests', [])).toBe(false)
  })
})

describe('outputProblem', () => {
  it('accepts a folder inside the workspace', () => {
    expect(outputProblem('docs/tests')).toBeUndefined()
    expect(outputProblem(String.raw` docs\api `)).toBeUndefined()
  })

  it('refuses the workspace root, an empty name and anything that leaves the workspace', () => {
    for (const bad of ['', '  ', '.', './']) {
      expect(outputProblem(bad)).toContain('Name a folder')
    }
    for (const bad of ['/etc', 'C:/out', String.raw`c:\out`, '..', '../out', 'a/../../b']) {
      expect(outputProblem(bad)).toContain('inside the workspace')
    }
  })
})
