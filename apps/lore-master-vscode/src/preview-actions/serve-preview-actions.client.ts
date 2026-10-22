import * as vscode from 'vscode'
import { ACTION_SCHEME, parsePreviewActionLink } from './preview-action-link.mapper'

/** What does the work for one named action. */
export type PreviewActionHandler = (params: Record<string, string>) => Promise<void> | void

/**
 * Serves the links LoreMaster puts in the Markdown preview. The preview opens such a link as a document; reading it runs the
 * handler of the action it names, and the tab it opened is closed again. A link for an action nobody handles does nothing.
 */
export function servePreviewActions (handlers: Record<string, PreviewActionHandler>): vscode.Disposable {
  const served = vscode.workspace.registerTextDocumentContentProvider(ACTION_SCHEME, {
    provideTextDocumentContent: uri => {
      const request = parsePreviewActionLink(uri)
      const handler = request === undefined ? undefined : handlers[request.action]
      if (request !== undefined && handler !== undefined) { void Promise.resolve(handler(request.params)).catch(() => {}) }

      return ''
    },
  })
  const closing = vscode.window.tabGroups.onDidChangeTabs(event => {
    for (const tab of event.opened) {
      if (tab.input instanceof vscode.TabInputText && tab.input.uri.scheme === ACTION_SCHEME) { void vscode.window.tabGroups.close(tab) }
    }
  })

  return vscode.Disposable.from(served, closing)
}
