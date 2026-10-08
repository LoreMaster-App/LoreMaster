import { isWatchedPath, workspacePath } from './watched-paths.policy'

describe('workspacePath', () => {
  it('is the path inside the folder, with forward slashes', () => {
    expect(workspacePath('/work/repo', '/work/repo/docs/guide.md')).toBe('docs/guide.md')
  })

  it('is undefined for the folder itself and for anything outside it', () => {
    expect(workspacePath('/work/repo', '/work/repo')).toBeUndefined()
    expect(workspacePath('/work/repo', '/work/other/a.md')).toBeUndefined()
    expect(workspacePath('/work/repo', '/work/repo/../a.md')).toBeUndefined()
  })
})

describe('isWatchedPath', () => {
  it.each([
    'README.md', 'docs/a.MD', '.lore-master.yaml', 'libs/core/a.go', 'src/app.ts', 'svc/main.py', 'reports/junit.xml',
    'src/Shop/Shop.csproj', 'pkg/lib/cart.dart', 'api/openapi.yaml',
  ])('watches %s', path => {
    expect(isWatchedPath(path)).toBe(true)
  })

  it.each([
    'node_modules/x/index.js', '.git/config', 'bin/Debug/a.json', 'assets/logo.png', '.hidden/a.md', 'obj/b.cs',
    'testdata/c.md', '.eslintrc.json', 'Makefile', 'docs/.draft.md', 'a/node_modules/b/c.md',
  ])('does not watch %s', path => {
    expect(isWatchedPath(path)).toBe(false)
  })
})
