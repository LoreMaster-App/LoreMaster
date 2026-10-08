import type { TreeNode } from '../engine-protocol'
import { fileNameOf, labelModeFrom, otherLabelMode, pageDescription, pageLabel } from './page-label.policy'

const node: TreeNode = { path: 'docs/architecture/engine.md', title: 'The engine', pageTitle: 'ENG: The engine', rule: 'selected-parent', depth: 1 }

describe('page labels', () => {
  it('shows the title in title mode, with the file name beside it', () => {
    expect(pageLabel(node, 'title')).toBe('The engine')
    expect(pageDescription(node, 'title')).toBe('engine.md')
  })

  it('shows the file name in file name mode, with the title beside it', () => {
    expect(pageLabel(node, 'fileName')).toBe('engine.md')
    expect(pageDescription(node, 'fileName')).toBe('The engine')
  })

  it('takes the name of a file at the root as it is', () => {
    expect(fileNameOf({ ...node, path: 'README.md' })).toBe('README.md')
  })

  it('reads a stored value, defaulting to title', () => {
    expect(labelModeFrom('fileName')).toBe('fileName')
    expect(labelModeFrom('title')).toBe('title')
    expect(labelModeFrom(undefined)).toBe('title')
    expect(labelModeFrom('nonsense')).toBe('title')
  })

  it('toggles between the two', () => {
    expect(otherLabelMode('title')).toBe('fileName')
    expect(otherLabelMode('fileName')).toBe('title')
  })
})
