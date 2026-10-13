import { decidePrepare, parsePrepareSetting } from './prepare-steps.policy'

const none = new Set<string>()

describe('parsePrepareSetting', () => {
  it('accepts the three values and falls back to ask', () => {
    expect(parsePrepareSetting('always')).toBe('always')
    expect(parsePrepareSetting('never')).toBe('never')
    expect(parsePrepareSetting('ask')).toBe('ask')
    expect(parsePrepareSetting(undefined)).toBe('ask')
    expect(parsePrepareSetting('sometimes')).toBe('ask')
  })
})

describe('decidePrepare', () => {
  const build = ['pnpm build:storybook']

  it('skips when no generator has a prepare command', () => {
    expect(decidePrepare({ commands: [], setting: 'always', trusted: true, approved: none })).toEqual({ kind: 'skip', reason: 'none' })
  })

  it('never runs anything with never, or in an untrusted workspace, whatever else is set', () => {
    expect(decidePrepare({ commands: build, setting: 'never', trusted: true, approved: none })).toEqual({ kind: 'skip', reason: 'never' })
    expect(decidePrepare({ commands: build, setting: 'always', trusted: false, approved: none })).toEqual({ kind: 'skip', reason: 'untrusted' })
  })

  it('runs at once with always in a trusted workspace', () => {
    expect(decidePrepare({ commands: build, setting: 'always', trusted: true, approved: none })).toEqual({ kind: 'run' })
  })

  it('asks with the exact commands under ask', () => {
    expect(decidePrepare({ commands: build, setting: 'ask', trusted: true, approved: none })).toEqual({ kind: 'ask', commands: build })
  })

  it('does not ask again for commands approved this session, but asks for a changed one', () => {
    const approved = new Set(build)
    expect(decidePrepare({ commands: build, setting: 'ask', trusted: true, approved })).toEqual({ kind: 'run' })
    expect(decidePrepare({ commands: [...build, 'rm -rf x'], setting: 'ask', trusted: true, approved }))
      .toEqual({ kind: 'ask', commands: [...build, 'rm -rf x'] })
  })
})
