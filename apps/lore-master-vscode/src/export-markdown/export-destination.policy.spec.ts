import * as path from 'node:path'
import { exportDestination } from './export-destination.policy'

const root = path.resolve('/work')

describe('exportDestination', () => {
  it('leaves the export next to the document when no folder is set', () => {
    expect(exportDestination(path.join(root, 'docs', 'a.md'), root, '')).toBeUndefined()
    expect(exportDestination(path.join(root, 'docs', 'a.md'), root, '  ')).toBeUndefined()
  })

  it('mirrors the document\'s place under the workspace inside the folder', () => {
    expect(exportDestination(path.join(root, 'docs', 'guides', 'a.md'), root, 'dist')).toBe(path.join(root, 'dist', 'docs', 'guides'))
    expect(exportDestination(path.join(root, 'a.md'), root, 'dist')).toBe(path.join(root, 'dist'))
  })

  it('takes an absolute folder as it is and keeps outside files out of the mirror', () => {
    const elsewhere = path.resolve('/exports')

    expect(exportDestination(path.join(root, 'docs', 'a.md'), root, elsewhere)).toBe(path.join(elsewhere, 'docs'))
    expect(exportDestination(path.resolve('/other/a.md'), root, 'dist')).toBe(path.join(root, 'dist'))
  })

  it('works without a workspace', () => {
    expect(exportDestination(path.resolve('/loose/a.md'), undefined, 'out')).toBe(path.join(path.resolve('/loose'), 'out'))
  })
})
