import { excludedModeFrom, otherExcludedMode, viewModeFrom } from './display-options.policy'

describe('display options', () => {
  it('reads a stored view mode, defaulting to the storage tree', () => {
    expect(viewModeFrom('repo')).toBe('repo')
    expect(viewModeFrom('flat')).toBe('flat')
    expect(viewModeFrom('nonsense')).toBe('storage')
    expect(viewModeFrom(undefined)).toBe('storage')
  })

  it('reads a stored excluded mode, defaulting to faded, and toggles', () => {
    expect(excludedModeFrom('hidden')).toBe('hidden')
    expect(excludedModeFrom(3)).toBe('faded')
    expect(otherExcludedMode('faded')).toBe('hidden')
    expect(otherExcludedMode('hidden')).toBe('faded')
  })
})
