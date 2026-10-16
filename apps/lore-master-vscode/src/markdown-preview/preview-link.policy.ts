import * as path from 'node:path'

/** What clicking a link in the preview does. */
export type PreviewLinkAction =
  | { kind: 'external', uri: string } |
  { kind: 'markdown', file: string, fragment: string } |
  { kind: 'ignore' }

const MARKDOWN_FILE = /\.(?:md|markdown)$/i
const SCHEME = /^[a-z][a-z\d+.-]*:/i

/**
 * Decides what a link does: an address on the web or a mail address opens outside, a link to another Markdown file opens that
 * file in the editor (at its heading, when the link has one), and anything else does nothing, so the preview cannot be
 * used to open arbitrary files.
 */
export function classifyLink (href: string, documentPath: string): PreviewLinkAction {
  if (/^(?:https?|mailto):/i.test(href)) {
    return { kind: 'external', uri: href }
  }
  if (SCHEME.test(href) || href.startsWith('//')) {
    return { kind: 'ignore' }
  }
  const [target, fragment = ''] = href.split('#', 2)
  let decoded: string
  try {
    decoded = decodeURIComponent(target)
  } catch {
    return { kind: 'ignore' }
  }
  if (!MARKDOWN_FILE.test(decoded)) {
    return { kind: 'ignore' }
  }

  return { kind: 'markdown', file: path.resolve(path.dirname(documentPath), decoded), fragment }
}
