import { afterMoves, mayHoldMarkdown, workspaceRelative } from './move-paths.algorithm'

describe('workspaceRelative', () => {
  it('makes a path relative to the folder, whatever the separators and case', () => {
    expect(workspaceRelative(String.raw`C:\Work\Repo`, String.raw`c:\work\repo\docs\a.md`)).toBe('docs/a.md')
    expect(workspaceRelative('/work/repo/', '/work/repo/a.md')).toBe('a.md')
  })

  it('is undefined outside the folder, and for a sibling that shares the prefix', () => {
    expect(workspaceRelative('/work/repo', '/work/other/a.md')).toBeUndefined()
    expect(workspaceRelative('/work/repo', '/work/repo-two/a.md')).toBeUndefined()
  })
})

describe('afterMoves', () => {
  const moves = [{ from: 'docs/a.md', to: 'docs/b.md' }, { from: 'old', to: 'new/deeper' }]

  it('follows a moved file and a file inside a moved folder', () => {
    expect(afterMoves('docs/a.md', moves)).toBe('docs/b.md')
    expect(afterMoves('old/x/y.md', moves)).toBe('new/deeper/x/y.md')
  })

  it('leaves everything else alone', () => {
    expect(afterMoves('docs/c.md', moves)).toBe('docs/c.md')
    expect(afterMoves('older/x.md', moves)).toBe('older/x.md')
  })
})

describe('mayHoldMarkdown', () => {
  it('accepts Markdown files and folders only', () => {
    expect(mayHoldMarkdown('docs/a.MD', false)).toBe(true)
    expect(mayHoldMarkdown('docs/v1.2', true)).toBe(true)
    expect(mayHoldMarkdown('docs/logo.png', false)).toBe(false)
  })
})
