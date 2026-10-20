import * as vscode from 'vscode'
import { addActionBar } from './add-action-bar.algorithm'
import { ACTION_SCHEME, parseActionLink } from './action-link.mapper'
import type { BarMarkdownIt, StatusEngine } from './action-bar.contract'
import { PageStatuses } from './page-status.store'

/** VS Code's own command that renders the open Markdown previews again. */
const REFRESH_PREVIEW_COMMAND = 'markdown.preview.refresh'

/** Answers that arrive together refresh the preview once. */
const REFRESH_AFTER_MS = 80

/** What the bar needs from the rest of the extension. */
export interface ActionBarDeps {
  engine:         StatusEngine
  /** Runs the export flow for a Markdown file. */
  exportDocument: (document: vscode.Uri) => Promise<void>
}

/**
 * The action bar of the built-in Markdown preview. The caller chains `extendMarkdownIt` onto the other preview extensions.
 * The Export button is a link the preview opens as a document, which this serves.
 */
export function registerActionBar (deps: ActionBarDeps): { extendMarkdownIt<Md extends BarMarkdownIt> (md: Md): Md } & vscode.Disposable {
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

  // The Export button is a link the preview opens as a document; reading that document does the work, and its tab is closed.
  const served = vscode.workspace.registerTextDocumentContentProvider(ACTION_SCHEME, {
    provideTextDocumentContent: uri => {
      const request = parseActionLink(uri)
      if (request !== undefined) { void deps.exportDocument(vscode.Uri.parse(request.document)) }

      return ''
    },
  })
  const closing = vscode.window.tabGroups.onDidChangeTabs(event => {
    for (const tab of event.opened) {
      if (tab.input instanceof vscode.TabInputText && tab.input.uri.scheme === ACTION_SCHEME) { void vscode.window.tabGroups.close(tab) }
    }
  })

  const watcher = vscode.workspace.onDidChangeConfiguration(event => {
    if (event.affectsConfiguration('loreMaster.preview')) {
      void vscode.commands.executeCommand(REFRESH_PREVIEW_COMMAND).then(undefined, () => {})
    }
  })

  return {
    extendMarkdownIt: md => addActionBar(md, statuses, enabled),
    dispose:          () => {
      clearTimeout(pending)
      for (const part of [served, closing, watcher]) { part.dispose() }
    },
  }
}
