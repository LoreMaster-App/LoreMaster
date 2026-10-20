import * as vscode from 'vscode'
import { addPageHeader } from './add-page-header.algorithm'
import type { HeaderEngine, HeaderMarkdownIt } from './page-header.contract'
import { PageHeaders } from './page-headers.store'

/** VS Code's own command that renders the open Markdown previews again. */
const REFRESH_PREVIEW_COMMAND = 'markdown.preview.refresh'

/** Answers that arrive together refresh the preview once. */
const REFRESH_AFTER_MS = 80

/**
 * The page header of the built-in Markdown preview: the title the file's page gets on each storage, and where it stands.
 * The caller chains `extendMarkdownIt` onto the other preview extensions.
 */
export function registerPageHeader (deps: { engine: HeaderEngine }): { extendMarkdownIt<Md extends HeaderMarkdownIt> (md: Md): Md } & vscode.Disposable {
  let pending: ReturnType<typeof setTimeout> | undefined
  const headers = new PageHeaders(
    deps.engine,
    documentPath => vscode.workspace.getWorkspaceFolder(vscode.Uri.file(documentPath))?.uri.fsPath,
    () => {
      clearTimeout(pending)
      pending = setTimeout(() => { void vscode.commands.executeCommand(REFRESH_PREVIEW_COMMAND).then(undefined, () => {}) }, REFRESH_AFTER_MS)
    },
  )

  return {
    extendMarkdownIt: md => addPageHeader(md, headers),
    dispose:          () => { clearTimeout(pending) },
  }
}
