import * as vscode from 'vscode'
import { type Output, SETTINGS_READ_METHOD, type TreeNode, WORKSPACE_TREE_METHOD, type WorkspaceTreeResult } from '../engine-protocol'
import type { LabelMode } from './page-tree.contract'
import { type DisplayOptions, type PagesEngine, type PagesNode, PagesViewProvider } from './pages-view.client'

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
        const asked = params as { output: number; scope?: string }
        const tree = trees[asked.scope === 'local' ? -1 : asked.output]

        return tree ? Promise.resolve(tree as never) : Promise.reject(new Error('settings are invalid'))
      }

      return Promise.resolve(null as never)
    },
  }
}

function setFolders (folders: { uri: { fsPath: string }; name: string }[] | undefined): void {
  (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = folders
}

function provider (
  outputs: Output[], trees: Record<number, WorkspaceTreeResult>, mode: LabelMode = 'title',
  requests: { method: string; params: unknown }[] = [], options: Partial<DisplayOptions> = {},
): PagesViewProvider {
  return new PagesViewProvider(engine(outputs, trees, requests), () => ({ label: mode, view: 'storage', excluded: 'faded', ...options }))
}

/** The rows under a storage node: the root also holds the Local node. */
const storageRows = async (view: PagesViewProvider, which = 0): Promise<PagesNode[]> => {
  const roots = await view.getChildren()

  return view.getChildren(roots.filter(row => row.kind === 'storage')[which])
}

const labels = async (view: PagesViewProvider, parent?: PagesNode): Promise<string[]> => {
  const rows = parent ? await view.getChildren(parent) : await storageRows(view)

  return rows.map(row => String(view.getTreeItem(row).label))
}

describe('PagesViewProvider', () => {
  beforeEach(() => { setFolders([{ uri: { fsPath: '/w' }, name: 'w' }]) })
  afterEach(() => { setFolders(undefined) })

  it('lists the files left out under a collapsed group, each with the rule that left it out', async () => {
    const view = provider([confluence], {
      0: {
        nodes,
        leftOut: [
          { path: 'CLAUDE.md', rule: 'ignore', pattern: 'CLAUDE.md' },
          { path: 'build/x.md', rule: 'gitignore', pattern: 'build/', source: '.gitignore' },
          { path: 'notes/y.md', rule: 'outside-roots' },
        ],
        leftOutTotal: 4,
      },
    })

    const rows = await storageRows(view)
    const group = rows.find(row => row.kind === 'left-out')!
    const item = view.getTreeItem(group)
    const files = await view.getChildren(group)

    expect(item.label).toBe('Left out (4)')
    expect(files.map(row => view.getTreeItem(row).label)).toEqual(['CLAUDE.md', 'build/x.md', 'notes/y.md', '… and 1 more'])
    expect(files.slice(0, 3).map(row => view.getTreeItem(row).description)).toEqual(['ignore: CLAUDE.md', '.gitignore: build/', 'outside the folders to sync'])
  })

  it('shows no left-out group when nothing is left out', async () => {
    const rows = await storageRows(provider([confluence], { 0: { nodes } }))

    expect(rows.some(row => row.kind === 'left-out')).toBe(false)
  })

  it('shows nothing without a workspace folder or a configured storage', async () => {
    setFolders(undefined)
    expect(await provider([confluence], { 0: { nodes } }).getChildren()).toEqual([])
    setFolders([{ uri: { fsPath: '/w' }, name: 'w' }])
    expect(await provider([scaffold], { 0: { nodes } }).getChildren()).toEqual([])
  })

  it('nests the pages of a storage as the sync would', async () => {
    const view = provider([confluence], { 0: { nodes } })

    expect(await labels(view)).toEqual(['Home', 'Guide'])
    const [home] = await storageRows(view)
    expect(await labels(view, home)).toEqual(['Setup'])
  })

  it('shows the Local node first and one group per storage', async () => {
    const view = provider([confluence, second], { 0: { nodes }, 1: { nodes: [node('a.md', 'A')] } })

    const groups = await view.getChildren()
    expect(groups.map(group => group.kind)).toEqual(['local', 'storage', 'storage'])
    expect(groups.map(group => String(view.getTreeItem(group).label))).toEqual(['Local', 'Confluence · ENG', 'Confluence · OPS'])
    expect(await labels(view, groups[2])).toEqual(['A'])
  })

  it('asks the engine for each output with the workspace folder, once until a refresh', async () => {
    const requests: { method: string; params: unknown }[] = []
    const view = provider([confluence], { 0: { nodes } }, 'title', requests)

    await storageRows(view)
    await storageRows(view)
    expect(requests.filter(request => request.method === WORKSPACE_TREE_METHOD)).toEqual([{ method: WORKSPACE_TREE_METHOD, params: { workspaceRoot: '/w', output: 0 } }])

    view.refresh()
    await storageRows(view)
    expect(requests.filter(request => request.method === WORKSPACE_TREE_METHOD)).toHaveLength(2)
  })

  it('names pages by title or by file name, with the other one beside it', async () => {
    const byTitle = provider([confluence], { 0: { nodes } }, 'title')
    const [home] = await storageRows(byTitle)
    expect(byTitle.getTreeItem(home).label).toBe('Home')
    expect(byTitle.getTreeItem(home).description).toBe('README.md')

    const byFile = provider([confluence], { 0: { nodes } }, 'fileName')
    expect(await labels(byFile)).toEqual(['guide.md', 'README.md'])
    const [first] = await storageRows(byFile)
    expect(byFile.getTreeItem(first).description).toBe('Guide')
  })

  it('draws an icon per status, and says in the tooltip that the platform was not checked', async () => {
    const view = provider([confluence], { 0: { nodes } })
    const [home, guide] = await storageRows(view)

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
    const [, guide] = await storageRows(view)
    const command = view.getTreeItem(guide).command

    expect(command?.command).toBe('vscode.open')
    expect((command?.arguments?.[0] as { fsPath: string }).fsPath).toMatch(/docs[/\\]guide\.md$/)
  })

  it('folds the platform check into the icons, lists orphans, and forgets it when the files change', async () => {
    const view = provider([confluence], { 0: { nodes } })
    view.setRemote(0, { actions: { 'README.md': 'conflict', 'docs/guide.md': 'conflict' }, orphans: [{ title: 'Old page', url: 'https://x/9' }] })

    const rows = await storageRows(view)
    const icons = rows.filter(row => row.kind === 'page').map(row => (view.getTreeItem(row).iconPath as { id: string }).id)
    expect(icons).toEqual(['cloud-download', 'warning'])
    expect(String(view.getTreeItem(rows[0]).tooltip)).not.toContain('has not been checked')
    expect(rows.at(-1)).toMatchObject({ kind: 'orphan', title: 'Old page' })
    expect(view.getTreeItem(rows.at(-1) as PagesNode).description).toBe('no file any more')

    view.redraw()
    const redrawn = await storageRows(view)
    expect(redrawn.some(row => row.kind === 'orphan')).toBe(true)
    view.refresh()
    const refreshed = await storageRows(view)
    expect(refreshed.some(row => row.kind === 'orphan')).toBe(false)
  })

  it('shows a page of an output that does not track pages with a plain icon', async () => {
    const pages: Output = { ...confluence, platform: 'github-pages', baseUrl: '', space: '' }
    const view = provider([pages], { 0: { nodes: [node('README.md', 'Home')] } })
    const [home] = await storageRows(view)

    expect((view.getTreeItem(home).iconPath as { id: string }).id).toBe('markdown')
    expect(view.getTreeItem(home).contextValue).toBe('loreMasterPageUntracked')
  })

  it('reports the engine\'s problems and failures as rows instead of throwing', async () => {
    const withProblem = provider([confluence], { 0: { nodes, problems: ['bad.md: could not be read'] } })
    expect(await labels(withProblem)).toEqual(['bad.md: could not be read', 'Home', 'Guide'])

    const failing = provider([confluence], {})
    expect(await labels(failing)).toEqual(['settings are invalid'])
  })

  describe('Local, arrangement and left-out files', () => {
    const local: WorkspaceTreeResult = {
      nodes: [
        { ...node('README.md', 'Home'), syncedTo: [0, 1] },
        { ...node('docs/guide.md', 'Guide'), syncedTo: [0] },
        { ...node('notes/plan.md', 'Plan') },
        { ...node('scratch.md', 'Scratch'), gitIgnored: true },
      ],
    }
    const trees = {
      0: { nodes, leftOut: [{ path: 'internal/faq.md', rule: 'excludes', pattern: 'internal/' }, { path: 'build/x.md', rule: 'gitignore', pattern: 'build/', source: '.gitignore' }] },
      1: { nodes: [node('a.md', 'A')] },
    }

    it('asks the engine for the local scope and says where each file syncs', async () => {
      const requests: { method: string; params: unknown }[] = []
      const view = provider([confluence, second], { ...trees, [-1]: local }, 'title', requests)
      const [localNode] = await view.getChildren()
      const rows = await view.getChildren(localNode)

      expect(requests.find(request => (request.params as { scope?: string }).scope === 'local')?.params).toEqual({ workspaceRoot: '/w', output: 0, scope: 'local' })
      expect(rows.map(row => view.getTreeItem(row).description)).toEqual([
        'README.md · → Confluence · ENG, Confluence · OPS',
        'guide.md · → Confluence · ENG',
        'plan.md · not synced',
        'scratch.md · ignored by git, never synced',
      ])
      expect(view.getTreeItem(rows[3]).contextValue).toBe('loreMasterIgnoredFile')
      expect(view.getTreeItem(rows[0]).contextValue).toBe('loreMasterPageUntracked')
    })

    it('shows a flat list ordered by path, without nesting', async () => {
      const view = provider([confluence], { 0: { nodes } }, 'title', [], { view: 'flat' })

      expect(await labels(view)).toEqual(['Guide', 'Home', 'Setup'])
      const rows = await storageRows(view)
      expect(rows.every(row => view.getTreeItem(row).collapsibleState === vscode.TreeItemCollapsibleState.None)).toBe(true)
    })

    it('shows the repository tree with the left-out files in place', async () => {
      const view = provider([confluence], trees, 'title', [], { view: 'repo' })
      const rows = await storageRows(view)

      expect(rows.map(row => `${row.kind}:${String(view.getTreeItem(row).label)}`)).toEqual([
        'folder:build', 'folder:docs', 'folder:internal', 'page:Home', 'page:Setup',
      ])
      const internal = rows.find(row => row.kind === 'folder' && row.folder.path === 'internal')!
      const [faq] = await view.getChildren(internal)
      expect(view.getTreeItem(faq).label).toBe('faq.md')
      expect(view.getTreeItem(faq).contextValue).toBe('loreMasterLeftOutFile')
      const build = rows.find(row => row.kind === 'folder' && row.folder.path === 'build')!
      const [ignored] = await view.getChildren(build)
      expect(view.getTreeItem(ignored).contextValue).toBe('loreMasterIgnoredFile')
    })

    it('hides the files a storage leaves out when asked to', async () => {
      const view = provider([confluence], trees, 'title', [], { excluded: 'hidden' })

      const rows = await storageRows(view)
      expect(rows.some(row => row.kind === 'left-out')).toBe(false)
    })
  })
})
