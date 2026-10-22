/**
 * A Markdown preview cannot run a command from a button (it enables no command links), but it opens a clicked link of any other
 * scheme as a document. LoreMaster's buttons and checkboxes in the preview are links of this scheme; the extension serves them as
 * an empty document that does the work as it is read, and closes the tab it opened.
 */
export const ACTION_SCHEME = 'loremaster-action'

/** What a link in the preview asks for: a named action and its parameters. */
export interface PreviewActionRequest {
  action: string
  params: Record<string, string>
}

/** The link that asks for an action. */
export function previewActionLink (action: string, params: Record<string, string>): string {
  const query = Object.entries(params).map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`).join('&')

  return `${ACTION_SCHEME}:/${encodeURIComponent(action)}${query === '' ? '' : `?${query}`}`
}

/** What a link asks for, or nothing when it is not one of LoreMaster's action links. */
export function parsePreviewActionLink (link: { scheme: string, path: string, query: string }): PreviewActionRequest | undefined {
  if (link.scheme !== ACTION_SCHEME || !link.path.startsWith('/') || link.path.length < 2) { return undefined }

  return { action: decodeURIComponent(link.path.slice(1)), params: Object.fromEntries(new URLSearchParams(link.query)) }
}
