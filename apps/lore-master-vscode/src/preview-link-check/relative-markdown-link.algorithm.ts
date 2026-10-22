import { dirname, resolve } from 'node:path'

const SCHEME = /^[a-z][\d+.a-z-]*:/i

/**
 * The file a link points at, when the link is a relative link to another Markdown file; nothing for an address with a scheme, a
 * link within the page, or a link to anything but a `.md` file. A fragment or query on the link is not part of the file.
 */
export function markdownTargetOf (href: string, documentPath: string): string | undefined {
  if (href === '' || href.startsWith('#') || href.startsWith('//') || SCHEME.test(href)) { return undefined }
  const [withoutQuery] = href.split(/[#?]/, 1)
  let decoded = withoutQuery
  try {
    decoded = decodeURIComponent(withoutQuery)
  } catch {
    // A link that is not valid percent-encoding is taken as written.
  }

  return decoded.toLowerCase().endsWith('.md') ? resolve(dirname(documentPath), decoded) : undefined
}
