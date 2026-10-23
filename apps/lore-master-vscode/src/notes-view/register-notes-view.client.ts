import * as vscode from 'vscode'
import { NOTES_SCAN_METHOD, type NotesScanResult } from '../engine-protocol'
import { NOTES_VIEW_ID, NotesViewProvider } from './notes-view.client'
import type { FoundNote, NotesGroupBy, NotesScan } from './notes.contract'

/** The command ids contributed in package.json. */
export const REFRESH_NOTES_COMMAND = 'loreMaster.refreshNotes'
const GROUP_BY_FILE_COMMAND = 'loreMaster.notesByFile'
const GROUP_BY_TAG_COMMAND = 'loreMaster.notesByTag'

/** How long to wait after a save or a file change before scanning again. */
const REFRESH_DEBOUNCE_MS = 800

/** The one thing the view needs from the engine client: to make a request. */
export interface NotesEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

const settings = (): vscode.WorkspaceConfiguration => vscode.workspace.getConfiguration('loreMaster.notes')

const groupBy = (): NotesGroupBy => settings().get<string>('groupBy') === 'file' ? 'file' : 'tag'

const tags = (): string[] => settings().get<string[]>('tags', []).map(tag => tag.trim()).filter(tag => tag !== '')

/** Scans every workspace folder with the engine; a folder that fails is reported and the others still count. */
async function scanFolders (engine: NotesEngine): Promise<NotesScan> {
  const folders = vscode.workspace.workspaceFolders ?? []
  const notes: FoundNote[] = []
  const problems: string[] = []
  let used: string[] = tags()
  let filesScanned = 0
  const answers = await Promise.allSettled(folders.map(folder => engine.request<NotesScanResult>(NOTES_SCAN_METHOD, { workspaceRoot: folder.uri.fsPath, tags: tags() })))
  for (const [index, answer] of answers.entries()) {
    if (answer.status === 'rejected') {
      problems.push(`${folders[index].name}: ${answer.reason instanceof Error ? answer.reason.message : String(answer.reason)}`)
      continue
    }
    used = answer.value.tags
    filesScanned += answer.value.filesScanned
    notes.push(...answer.value.notes.map(note => ({ ...note, folder: folders[index].uri.fsPath })))
  }

  return { notes, tags: used, filesScanned, problems }
}

/**
 * Registers the Notes view and what keeps it current: a scan when it is first shown and after files are saved, created, deleted or
 * renamed (a short pause after the last change), and when its settings change. While the view is hidden nothing is scanned; it
 * scans when it is shown again.
 */
export function registerNotesView (deps: { engine: NotesEngine }): vscode.Disposable {
  // The count on the view's icon is set when a scan ends, which is always after the view below exists.
  const provider = new NotesViewProvider({ scan: () => scanFolders(deps.engine), groupBy }, scan => {
    treeView.badge = scan.notes.length > 0 ? { value: scan.notes.length, tooltip: `${scan.notes.length} notes` } : undefined
  })
  const treeView = vscode.window.createTreeView(NOTES_VIEW_ID, { treeDataProvider: provider, showCollapseAll: true })

  let stale = false
  let timer: ReturnType<typeof setTimeout> | undefined
  const changed = (): void => {
    if (!treeView.visible) {
      stale = true

      return
    }
    clearTimeout(timer)
    timer = setTimeout(() => provider.refresh(), REFRESH_DEBOUNCE_MS)
  }
  const setGroupBy = async (value: NotesGroupBy): Promise<void> => {
    await settings().update('groupBy', value, vscode.ConfigurationTarget.Global)
    provider.redraw()
  }

  return vscode.Disposable.from(
    treeView,
    vscode.commands.registerCommand(REFRESH_NOTES_COMMAND, () => provider.refresh()),
    vscode.commands.registerCommand(GROUP_BY_FILE_COMMAND, () => setGroupBy('file')),
    vscode.commands.registerCommand(GROUP_BY_TAG_COMMAND, () => setGroupBy('tag')),
    treeView.onDidChangeVisibility(event => {
      if (!stale || !event.visible) {
        return
      }

      stale = false
      provider.refresh()
    }),
    vscode.workspace.onDidSaveTextDocument(changed),
    vscode.workspace.onDidCreateFiles(changed),
    vscode.workspace.onDidDeleteFiles(changed),
    vscode.workspace.onDidRenameFiles(changed),
    vscode.workspace.onDidChangeWorkspaceFolders(changed),
    vscode.workspace.onDidChangeConfiguration(event => {
      if (event.affectsConfiguration('loreMaster.notes.tags')) {
        changed()
      } else if (event.affectsConfiguration('loreMaster.notes.groupBy')) {
        provider.redraw()
      }
    }),
    { dispose: () => { clearTimeout(timer) } },
  )
}
