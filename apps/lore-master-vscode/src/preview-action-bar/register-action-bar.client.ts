import * as vscode from 'vscode'
import { addActionBar } from './add-action-bar.algorithm'
import type { BarMarkdownIt, PageStatusLookup, StatusEngine } from './action-bar.contract'
import { PageStatuses } from './page-status.store'

/** VS Code's own command that renders the open Markdown previews again. */
const REFRESH_PREVIEW_COMMAND = 'markdown.preview.refresh'

/** Answers that arrive together refresh the preview once. */
const REFRESH_AFTER_MS = 80

/**
 * The action bar of the built-in Markdown preview. The caller chains `extendMarkdownIt` onto the other preview extensions.
 * Its buttons are links the preview opens as documents; `servePreviewActions` answers them.
 */
export function registerActionBar (deps: { engine: StatusEngine }): { extendMarkdownIt<Md extends BarMarkdownIt> (md: Md): Md, statusOf: PageStatusLookup['statusOf'] } & vscode.Disposable {
  let pending: ReturnType<typeof setTimeout> | undefined
  const statuses = new PageStatuses(
    deps.engine,
    documentPath => vscode.workspace.getWorkspaceFolder(vscode.Uri.file(documentPath))?.uri.fsPath,
    () => {
      clearTimeout(pending)
      pending = setTimeout(() => { void vscode.commands.executeCommand(REFRESH_PREVIEW_COMMAND).then(undefined, () => {}) }, REFRESH_AFTER_MS)
    },
  )
  const enabled = (): boolean => vscode.workspace.getConfiguration('loreMaster.preview').get<boolean>('actionBar', true)

  const watcher = vscode.workspace.onDidChangeConfiguration(event => {
    if (event.affectsConfiguration('loreMaster.preview')) {
      void vscode.commands.executeCommand(REFRESH_PREVIEW_COMMAND).then(undefined, () => {})
    }
  })

  return {
    extendMarkdownIt: md => addActionBar(md, statuses, enabled),
    statusOf:         documentPath => statuses.statusOf(documentPath),
    dispose:          () => {
      clearTimeout(pending)
      watcher.dispose()
    },
  }
}
