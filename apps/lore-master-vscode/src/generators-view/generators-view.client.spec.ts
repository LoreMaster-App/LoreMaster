import * as vscode from 'vscode'
import { type Generator, type GeneratorRun, SETTINGS_READ_METHOD } from '../engine-protocol'
import { GeneratorsViewProvider } from './generators-view.client'
import type { GeneratorsViewEngine } from './generators.use-case'

const tests: Generator = { type: 'test-results', output: 'docs/tests', input: ['reports/', '**/junit*.xml'], title: 'CI results' }
const api: Generator = { type: 'go-docs', output: 'docs/api' }

function engine (generators: Generator[] | undefined, fail?: boolean): GeneratorsViewEngine {
  return {
    request (method: string) {
      if (method === SETTINGS_READ_METHOD && fail) {
        return Promise.reject(new Error('invalid settings'))
      }

      return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs: [], generators } } as never)
    },
  }
}

function setFolders (folders: { uri: { fsPath: string }; name: string }[] | undefined): void {
  (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = folders
}

const run = (overrides: Partial<GeneratorRun>): GeneratorRun => ({ index: 0, type: 'test-results', output: 'docs/tests', ...overrides })

describe('GeneratorsViewProvider', () => {
  beforeEach(() => { setFolders([{ uri: { fsPath: '/w' }, name: 'w' }]) })
  afterEach(() => { setFolders(undefined) })

  it('lists one row per configured generator, in order', async () => {
    const view = new GeneratorsViewProvider(engine([tests, api]))

    const rows = await view.getChildren()

    expect(rows.map(row => row.index)).toEqual([0, 1])
    expect(rows.map(row => view.getTreeItem(row).label)).toEqual(['Test results', 'Go package docs'])
  })

  it('shows nothing without a folder, without generators, or when the settings cannot be read', async () => {
    setFolders(undefined)
    expect(await new GeneratorsViewProvider(engine([tests])).getChildren()).toEqual([])
    setFolders([{ uri: { fsPath: '/w' }, name: 'w' }])
    expect(await new GeneratorsViewProvider(engine(undefined)).getChildren()).toEqual([])
    expect(await new GeneratorsViewProvider(engine([tests], true)).getChildren()).toEqual([])
  })

  it('has no children under a row', async () => {
    const view = new GeneratorsViewProvider(engine([tests]))
    const [row] = await view.getChildren()

    expect(await view.getChildren(row)).toEqual([])
  })

  it('describes a row by where it writes, with an icon for its kind and the details in the tooltip', async () => {
    const view = new GeneratorsViewProvider(engine([tests, api]))
    const [first, second] = await view.getChildren()

    const item = view.getTreeItem(first)
    expect(item.description).toBe('→ docs/tests')
    expect((item.iconPath as { id: string }).id).toBe('beaker')
    expect(item.contextValue).toBe('loreMasterGenerator')
    expect(String(item.tooltip)).toContain('Writes to: docs/tests')
    expect(String(item.tooltip)).toContain('Reads: reports/, **/junit*.xml')
    expect(String(item.tooltip)).toContain('Index page title: CI results')
    expect(String(view.getTreeItem(second).tooltip)).toContain('Reads: the default')
    expect((view.getTreeItem(second).iconPath as { id: string }).id).toBe('package')
  })

  it('keeps the name of a generator type it does not know', () => {
    const view = new GeneratorsViewProvider(engine([]))
    const item = view.getTreeItem({ index: 0, generator: { type: 'future-docs', output: 'docs/f' } })

    expect(item.label).toBe('future-docs')
    expect((item.iconPath as { id: string }).id).toBe('question')
  })

  it('shows what a run did on the rows it ran, until the configuration changes', async () => {
    const view = new GeneratorsViewProvider(engine([tests, api]))
    const rows = await view.getChildren()

    view.recordRun({ runs: [run({ written: ['a.md', 'b.md'], unchanged: ['c.md'] }), run({ index: 1, error: 'no packages' })] })

    expect(view.getTreeItem(rows[0]).description).toBe('→ docs/tests · 2 written, 1 unchanged, 0 removed')
    expect(view.getTreeItem(rows[1]).description).toBe('→ docs/api · failed: no packages')
    expect(String(view.getTreeItem(rows[0]).tooltip)).toContain('Last run: 2 written')

    view.refresh()

    expect(view.getTreeItem(rows[0]).description).toBe('→ docs/tests')
  })

  it('tells the tree to redraw after a refresh and after a run', () => {
    const view = new GeneratorsViewProvider(engine([tests]))
    let redraws = 0
    view.onDidChangeTreeData(() => { redraws++ })

    view.refresh()
    view.recordRun({ runs: [] })

    expect(redraws).toBe(2)
  })
})
