import * as vscode from 'vscode'
import { type Output, SETTINGS_READ_METHOD, type TreeNode, WORKSPACE_TREE_METHOD, type WorkspaceTreeResult } from '../engine-protocol'
import type { LabelMode } from './page-tree.contract'
import { type PagesEngine, type PagesNode, PagesViewProvider } from './pages-view.client'

const confluence: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}
const second: Output = { ...confluence, space: 'OPS' }
const scaffold: Output = { ...confluence, baseUrl: '', space: '', parentPageId: '', titlePrefix: '' }

function node (path: string, title: string, parent?: string, status?: TreeNode['status']): TreeNode {
  return { path, title, pageTitle: `ENG: ${title}`, parent, rule: 'selected-parent', depth: 0, status }
}

const nodes = [
  node('README.md', 'Home', undefined, 'synced'),
  node('readme.setup.md', 'Setup', 'README.md', 'new'),
  node('docs/guide.md', 'Guide', undefined, 'local-changes'),
]

function engine (outputs: Output[], trees: Record<number, WorkspaceTreeResult>, requests: { method: string; params: unknown }[] = []): PagesEngine {
  return {
    request (method: string, params?: unknown) {
      requests.push({ method, params })
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs } } as never)
      }
      if (method === WORKSPACE_TREE_METHOD) {
        const tree = trees[(params as { output: number }).output]

        return tree ? Promise.resolve(tree as never) : Promise.reject(new Error('settings are invalid'))
      }

      return Promise.resolve(null as never)
    },
  }
}

function setFolders (folders: { uri: { fsPath: string }; name: string }[] | undefined): void {
  (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = folders
}

function provider (outputs: Output[], trees: Record<number, WorkspaceTreeResult>, mode: LabelMode = 'title', requests: { method: string; params: unknown }[] = []): PagesViewProvider {
  return new PagesViewProvider(engine(outputs, trees, requests), () => mode)
}

const labels = async (view: PagesViewProvider, parent?: PagesNode): Promise<string[]> => {
  const rows = await view.getChildren(parent)

  return rows.map(row => String(view.getTreeItem(row).label))
}

describe('PagesViewProvider', () => {
  beforeEach(() => { setFolders([{ uri: { fsPath: '/w' }, name: 'w' }]) })
  afterEach(() => { setFolders(undefined) })

  it('shows nothing without a workspace folder or a configured storage', async () => {
    setFolders(undefined)
    expect(await provider([confluence], { 0: { nodes } }).getChildren()).toEqual([])
    setFolders([{ uri: { fsPath: '/w' }, name: 'w' }])
    expect(await provider([scaffold], { 0: { nodes } }).getChildren()).toEqual([])
  })

  it('lists the pages directly when there is one storage, nested as the sync would', async () => {
    const view = provider([confluence], { 0: { nodes } })

    expect(await labels(view)).toEqual(['Home', 'Guide'])
    const [home] = await view.getChildren()
    expect(await labels(view, home)).toEqual(['Setup'])
  })

  it('shows one group per storage when there are several', async () => {
    const view = provider([confluence, second], { 0: { nodes }, 1: { nodes: [node('a.md', 'A')] } })

    const groups = await view.getChildren()
    expect(groups.map(group => group.kind)).toEqual(['storage', 'storage'])
    expect(await labels(view)).toEqual(['Confluence · ENG', 'Confluence · OPS'])
    expect(await labels(view, groups[1])).toEqual(['A'])
  })

  it('asks the engine for each output with the workspace folder, once until a refresh', async () => {
    const requests: { method: string; params: unknown }[] = []
    const view = provider([confluence], { 0: { nodes } }, 'title', requests)

    await view.getChildren()
    await view.getChildren()
    expect(requests.filter(request => request.method === WORKSPACE_TREE_METHOD)).toEqual([{ method: WORKSPACE_TREE_METHOD, params: { workspaceRoot: '/w', output: 0 } }])

    view.refresh()
    await view.getChildren()
    expect(requests.filter(request => request.method === WORKSPACE_TREE_METHOD)).toHaveLength(2)
  })

  it('names pages by title or by file name, with the other one beside it', async () => {
    const byTitle = provider([confluence], { 0: { nodes } }, 'title')
    const [home] = await byTitle.getChildren()
    expect(byTitle.getTreeItem(home).label).toBe('Home')
    expect(byTitle.getTreeItem(home).description).toBe('README.md')

    const byFile = provider([confluence], { 0: { nodes } }, 'fileName')
    expect(await labels(byFile)).toEqual(['guide.md', 'README.md'])
    const [first] = await byFile.getChildren()
    expect(byFile.getTreeItem(first).description).toBe('Guide')
  })

  it('draws an icon per status, and says in the tooltip that the platform was not checked', async () => {
    const view = provider([confluence], { 0: { nodes } })
    const [home, guide] = await view.getChildren()

    expect((view.getTreeItem(home).iconPath as { id: string }).id).toBe('check')
    expect((view.getTreeItem(guide).iconPath as { id: string }).id).toBe('edit')
    expect(String(view.getTreeItem(guide).tooltip)).toContain('Local changes')
    expect(String(view.getTreeItem(guide).tooltip)).toContain('has not been checked')
    expect(String(view.getTreeItem(guide).tooltip)).toContain('docs/guide.md')
    const children = await view.getChildren(home)
    expect((view.getTreeItem(children[0]).iconPath as { id: string }).id).toBe('diff-added')
  })

  it('opens the file when a page is clicked', async () => {
    const view = provider([confluence], { 0: { nodes } })
    const [, guide] = await view.getChildren()
    const command = view.getTreeItem(guide).command

    expect(command?.command).toBe('vscode.open')
    expect((command?.arguments?.[0] as { fsPath: string }).fsPath).toMatch(/docs[/\\]guide\.md$/)
  })

  it('folds the platform check into the icons, lists orphans, and forgets it when the files change', async () => {
    const view = provider([confluence], { 0: { nodes } })
    view.setRemote(0, { actions: { 'README.md': 'conflict', 'docs/guide.md': 'conflict' }, orphans: [{ title: 'Old page', url: 'https://x/9' }] })

    const rows = await view.getChildren()
    const icons = rows.filter(row => row.kind === 'page').map(row => (view.getTreeItem(row).iconPath as { id: string }).id)
    expect(icons).toEqual(['cloud-download', 'warning'])
    expect(String(view.getTreeItem(rows[0]).tooltip)).not.toContain('has not been checked')
    expect(rows.at(-1)).toMatchObject({ kind: 'orphan', title: 'Old page' })
    expect(view.getTreeItem(rows.at(-1) as PagesNode).description).toBe('no file any more')

    view.redraw()
    const redrawn = await view.getChildren()
    expect(redrawn.some(row => row.kind === 'orphan')).toBe(true)
    view.refresh()
    const refreshed = await view.getChildren()
    expect(refreshed.some(row => row.kind === 'orphan')).toBe(false)
  })

  it('shows a page of an output that does not track pages with a plain icon', async () => {
    const pages: Output = { ...confluence, platform: 'github-pages', baseUrl: '', space: '' }
    const view = provider([pages], { 0: { nodes: [node('README.md', 'Home')] } })
    const [home] = await view.getChildren()

    expect((view.getTreeItem(home).iconPath as { id: string }).id).toBe('markdown')
    expect(view.getTreeItem(home).contextValue).toBe('loreMasterPageUntracked')
  })

  it('reports the engine\'s problems and failures as rows instead of throwing', async () => {
    const withProblem = provider([confluence], { 0: { nodes, problems: ['bad.md: could not be read'] } })
    expect(await labels(withProblem)).toEqual(['bad.md: could not be read', 'Home', 'Guide'])

    const failing = provider([confluence], {})
    expect(await labels(failing)).toEqual(['settings are invalid'])
  })
})
