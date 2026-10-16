import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { EXPORT_FORMATS_METHOD, EXPORT_RUN_METHOD, type ExportRunParams } from '../engine-protocol'
import { copyMarkdownAsCommand, exportMarkdownCommand } from './export-markdown.handler'

const window = vscode.window as unknown as Record<string, unknown>
const virtualFiles = (vscode as unknown as { virtualFiles: Map<string, Uint8Array> }).virtualFiles
const original = { ...window }
const messages: string[] = []
const runs: ExportRunParams[] = []
const output = { appendLine () {}, show () {} } as unknown as vscode.OutputChannel

const engine = {
  request (method: string, params?: unknown) {
    if (method === EXPORT_FORMATS_METHOD) {
      return Promise.resolve({ formats: [{ id: 'html', label: 'HTML', extension: '.html' }, { id: 'jira-wiki', label: 'Jira wiki', extension: '.jira.txt' }, { id: 'docx', label: 'Word', extension: '.docx', binary: true }] } as never)
    }
    expect(method).toBe(EXPORT_RUN_METHOD)
    const run = params as ExportRunParams
    runs.push(run)

    return Promise.resolve(run.inline ? { name: 'a', text: `<<${run.content}>>` } as never : { name: 'a.html', path: `/w/${run.format}.out` } as never)
  },
} as unknown as EngineClient

beforeEach(() => {
  messages.length = 0
  runs.length = 0
  virtualFiles.clear()
  window.showInformationMessage = (message: string) => {
    messages.push(`info: ${message}`)

    return Promise.resolve(undefined)
  }
  window.showErrorMessage = (message: string) => {
    messages.push(`error: ${message}`)

    return Promise.resolve(undefined)
  }
  window.activeTextEditor = undefined
})

afterEach(() => {
  Object.assign(window, original)
})

const editor = (selected: string | undefined): unknown => ({
  document:  { languageId: 'markdown', uri: { fsPath: '/w/a.md' }, getText: (range?: unknown) => (range ? (selected ?? '') : 'whole document') },
  selection: { isEmpty: selected === undefined },
})

describe('copyMarkdownAsCommand', () => {
  it('copies the whole document in the chosen format', async () => {
    window.activeTextEditor = editor(undefined)
    window.showQuickPick = (items: { id: string }[]) => Promise.resolve(items.find(item => item.id === 'jira-wiki'))

    await copyMarkdownAsCommand({ engine, output })

    expect(await vscode.env.clipboard.readText()).toBe('<<whole document>>')
    expect(runs[0]).toMatchObject({ format: 'jira-wiki', inline: true, path: '/w/a.md' })
  })

  it('copies just the selection when there is one, and offers text formats only', async () => {
    window.activeTextEditor = editor('picked lines')
    let offered: string[] = []
    window.showQuickPick = (items: { id: string }[]) => {
      offered = items.map(item => item.id)

      return Promise.resolve(items[0])
    }

    await copyMarkdownAsCommand({ engine, output })

    expect(offered).toEqual(['html', 'jira-wiki'])
    expect(await vscode.env.clipboard.readText()).toBe('<<picked lines>>')
  })

  it('says so when no Markdown file is open', async () => {
    await copyMarkdownAsCommand({ engine, output })

    expect(messages).toEqual(['info: LoreMaster: open a Markdown file to copy it.'])
  })
})

describe('exportMarkdownCommand', () => {
  it('says so when there is nothing to export', async () => {
    await exportMarkdownCommand({ engine, output })

    expect(messages[0]).toContain('open a Markdown file')
    expect(runs).toHaveLength(0)
  })

  it('exports the clicked file to every ticked format', async () => {
    virtualFiles.set('/w/a.md', new Uint8Array())
    window.showQuickPick = (items: { id: string }[]) => Promise.resolve(items.filter(item => item.id !== 'jira-wiki'))

    await exportMarkdownCommand({ engine, output }, vscode.Uri.file('/w/a.md'))

    expect(runs.map(run => run.format)).toEqual(['html', 'docx'])
    expect(messages).toEqual(['info: LoreMaster: exported 2 files.'])
  })

  it('does nothing when the picker is dismissed', async () => {
    virtualFiles.set('/w/a.md', new Uint8Array())
    window.showQuickPick = () => Promise.resolve(undefined)

    await exportMarkdownCommand({ engine, output }, vscode.Uri.file('/w/a.md'))

    expect(runs).toHaveLength(0)
  })
})
