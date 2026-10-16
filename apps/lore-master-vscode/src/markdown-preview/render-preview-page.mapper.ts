const escape = (text: string): string => text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')

/** The content security policy of the preview: nothing runs but its own script, and images are the embedded ones or from the web. */
function policy (nonce: string): string {
  return `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data: https: http:; style-src 'unsafe-inline'; font-src data:; script-src 'nonce-${nonce}'">`
}

/**
 * The page's own script: keeps the scroll position when the page is replaced, scrolls with the editor and tells the extension
 * how far it was scrolled, and hands clicks on links to the extension (except links within the page, which scroll as usual).
 */
function script (nonce: string): string {
  return `<script nonce="${nonce}">
const vscode = acquireVsCodeApi()
let quiet = false
const range = () => Math.max(document.documentElement.scrollHeight - window.innerHeight, 0)
const fraction = () => (range() > 0 ? window.scrollY / range() : 0)
function scrollToFraction (value) {
  quiet = true
  window.scrollTo(0, value * range())
  vscode.setState({ fraction: value })
  setTimeout(() => { quiet = false }, 120)
}
window.addEventListener('load', () => {
  const saved = vscode.getState()
  if (saved && typeof saved.fraction === 'number') { scrollToFraction(saved.fraction) }
  vscode.postMessage({ type: 'ready' })
})
window.addEventListener('scroll', () => {
  if (quiet) { return }
  vscode.setState({ fraction: fraction() })
  vscode.postMessage({ type: 'scroll', fraction: fraction() })
})
window.addEventListener('message', event => {
  if (event.data && event.data.type === 'scroll') { scrollToFraction(event.data.fraction) }
})
document.addEventListener('click', event => {
  const link = event.target instanceof Element ? event.target.closest('a') : null
  const href = link ? link.getAttribute('href') : null
  if (!href || href.startsWith('#')) { return }
  event.preventDefault()
  vscode.postMessage({ type: 'open', href })
})
</script>`
}

/** The preview of a document: the HTML the engine exported, with the policy and the script added. */
export function renderHtmlPreview (html: string, nonce: string): string {
  const head = html.indexOf('<head>')
  const withPolicy = head === -1
    ? policy(nonce) + html
    : `${html.slice(0, head + 6)}
${policy(nonce)}${html.slice(head + 6)}`
  // The last one, in case the document itself writes `</body>`.
  const body = withPolicy.lastIndexOf('</body>')

  return body === -1
    ? withPolicy + script(nonce)
    : `${withPolicy.slice(0, body)}${script(nonce)}
${withPolicy.slice(body)}`
}

/** The preview of a document as text in another format (Jira wiki, MediaWiki...), shown as it will be pasted. */
export function renderTextPage (text: string, label: string, nonce: string): string {
  return `<!doctype html><html><head><meta charset="utf-8">${policy(nonce)}<style>
body{margin:0;padding:12px 16px;font:13px/1.5 var(--vscode-editor-font-family,monospace);color:var(--vscode-editor-foreground);background:var(--vscode-editor-background)}
h1{font:600 12px var(--vscode-font-family,sans-serif);opacity:.7;margin:0 0 8px}
pre{white-space:pre-wrap;word-break:break-word;margin:0}
</style></head><body><h1>${escape(label)}</h1><pre>${escape(text)}</pre></body></html>`
}
