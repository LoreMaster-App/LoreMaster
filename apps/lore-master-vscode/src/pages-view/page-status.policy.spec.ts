import { combineStatus, statusPresentation } from './page-status.policy'

describe('combineStatus', () => {
  it.each([
    ['new', undefined, 'new'],
    ['synced', undefined, 'synced'],
    ['local-changes', undefined, 'local-changes'],
    ['synced', 'unchanged', 'synced'],
    ['synced', 'conflict', 'remote-changes'],
    ['local-changes', 'conflict', 'conflict'],
    ['synced', 'pull', 'remote-changes'],
    ['new', 'adopt', 'adopt'],
    ['synced', 'create', 'new'],
    ['synced', 'update', 'local-changes'],
    ['synced', 'move', 'local-changes'],
    ['synced', 'rename_title', 'local-changes'],
    ['synced', 'something-new', 'synced'],
  ] as const)('local %s with remote %s is %s', (local, remote, expected) => {
    expect(combineStatus(local, remote)).toBe(expected)
  })

  it('has no status for an output that does not track pages', () => {
    expect(combineStatus(undefined, 'update')).toBeUndefined()
  })
})

describe('statusPresentation', () => {
  it('describes every status with an icon, a colour and words', () => {
    for (const status of ['new', 'synced', 'local-changes', 'remote-changes', 'conflict', 'adopt'] as const) {
      const presentation = statusPresentation(status)
      expect(presentation.icon).not.toBe('')
      expect(presentation.color).not.toBe('')
      expect(presentation.label).not.toBe('')
      expect(presentation.detail).not.toBe('')
    }
  })
})
