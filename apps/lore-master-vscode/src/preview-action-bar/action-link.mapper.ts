/**
 * A Markdown preview cannot run a command from a button (it enables no command links), but it opens a clicked link of any other
 * scheme as a document. The bar's buttons are links of this scheme; the extension serves them as an empty document that does the
 * work as it is read, and closes the tab it opened.
 */
export const ACTION_SCHEME = 'loremaster-action'

/** What a bar button asks for. */
export interface ActionRequest {
  action:   'export'
  /** The Markdown file the preview shows, as a URI string. */
  document: string
}

/** The link the Export button carries. */
export function exportLink (document: string): string {
  return `${ACTION_SCHEME}:/export?document=${encodeURIComponent(document)}`
}

/** What a link asks for, or nothing when it is not one of the bar's links. */
export function parseActionLink (link: { scheme: string, path: string, query: string }): ActionRequest | undefined {
  if (link.scheme !== ACTION_SCHEME || link.path !== '/export') { return undefined }
  const document = new URLSearchParams(link.query).get('document')

  return document === null || document === '' ? undefined : { action: 'export', document }
}
