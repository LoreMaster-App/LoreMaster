import * as vscode from 'vscode'
import { folderForActiveEditor, pickWorkspaceFolder } from './workspace-fs.client'

// The spec resolves `vscode` to @types/vscode, where these are readonly; at runtime the
// jest moduleNameMapper points `vscode` at the mutable stub. This cast lets a test script
// what the stub returns, on the very object the slice reads.
interface MutableVscode {
  workspace: {
    workspaceFolders?:  { uri: { fsPath: string }; name: string }[]
    getWorkspaceFolder: (uri: unknown) => { uri: { fsPath: string } } | undefined
  }
  window: {
    showQuickPick:     (items: unknown, options?: unknown) => Promise<unknown>
    activeTextEditor?: { document: { uri: unknown } }
  }
}
const stub = vscode as unknown as MutableVscode

const folder = (name: string, fsPath: string): { uri: { fsPath: string }; name: string } => ({ name, uri: { fsPath } })

describe('pickWorkspaceFolder', () => {
  afterEach(() => {
    stub.workspace.workspaceFolders = undefined
    stub.window.showQuickPick = () => Promise.resolve(undefined)
  })

  it('returns undefined when no folder is open', async () => {
    stub.workspace.workspaceFolders = undefined

    expect(await pickWorkspaceFolder()).toBeUndefined()
  })

  it('returns the only folder without prompting', async () => {
    stub.workspace.workspaceFolders = [folder('docs', '/w/docs')]
    let prompted = false
    stub.window.showQuickPick = () => {
      prompted = true

      return Promise.resolve(undefined)
    }

    expect(await pickWorkspaceFolder()).toBe('/w/docs')
    expect(prompted).toBe(false)
  })

  it('prompts a QuickPick when several folders are open, returning the chosen path', async () => {
    stub.workspace.workspaceFolders = [folder('a', '/w/a'), folder('b', '/w/b')]
    stub.window.showQuickPick = () => Promise.resolve({ label: 'b', description: '/w/b' })

    expect(await pickWorkspaceFolder()).toBe('/w/b')
  })

  it('returns undefined when the QuickPick is cancelled', async () => {
    stub.workspace.workspaceFolders = [folder('a', '/w/a'), folder('b', '/w/b')]
    stub.window.showQuickPick = () => Promise.resolve(undefined)

    expect(await pickWorkspaceFolder()).toBeUndefined()
  })
})

describe('folderForActiveEditor', () => {
  afterEach(() => {
    stub.window.activeTextEditor = undefined
    stub.workspace.getWorkspaceFolder = () => undefined
  })

  it('returns undefined with no active editor', () => {
    stub.window.activeTextEditor = undefined

    expect(folderForActiveEditor()).toBeUndefined()
  })

  it('returns the folder containing the active file', () => {
    stub.window.activeTextEditor = { document: { uri: { path: '/w/a/readme.md' } } }
    stub.workspace.getWorkspaceFolder = () => ({ uri: { fsPath: '/w/a' } })

    expect(folderForActiveEditor()).toBe('/w/a')
  })

  it('returns undefined when the file is outside the workspace', () => {
    stub.window.activeTextEditor = { document: { uri: { path: '/tmp/x.md' } } }
    stub.workspace.getWorkspaceFolder = () => undefined

    expect(folderForActiveEditor()).toBeUndefined()
  })
})
