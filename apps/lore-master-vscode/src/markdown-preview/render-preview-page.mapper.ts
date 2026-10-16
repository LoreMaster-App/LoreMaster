const escape = (text: string): string => text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')

/** The content security policy of the preview: nothing runs but its own script, and images are the embedded ones or from the web. */
function policy (nonce: string, cspSource = ''): string {
  const own = cspSource === '' ? '' : ` ${cspSource}`

  return `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src${own} data: https: http:; style-src 'unsafe-inline'${own}; font-src data:; script-src 'nonce-${nonce}'${own}">`
}

/** Where the bundled Mermaid library is served from, and the webview origin the page must allow for it. */
export interface MermaidSource {
  scriptUri: string
  cspSource: string
}

/**
 * The page's own script: draws the Mermaid diagrams, keeps the scroll position when the page is replaced, scrolls with the
 * editor and tells the extension how far it was scrolled, and hands clicks on links to the extension (except links within
 * the page, which scroll as usual).
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
async function drawDiagrams () {
  if (typeof mermaid === 'undefined') { return }
  const dark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
  mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: dark ? 'dark' : 'default' })
  let count = 0
  for (const block of document.querySelectorAll('pre > code.language-mermaid')) {
    const holder = document.createElement('div')
    holder.className = 'diagram'
    try {
      const { svg } = await mermaid.render('lore-master-diagram-' + count, block.textContent)
      holder.innerHTML = svg
    } catch (error) {
      holder.textContent = 'The diagram could not be drawn: ' + (error && error.message ? error.message : String(error))
      holder.style.color = '#c0392b'
      // Mermaid leaves its own error picture in the page when a diagram fails; the message above replaces it.
      for (const leftover of [document.getElementById('lore-master-diagram-' + count), document.getElementById('dlore-master-diagram-' + count)]) { if (leftover) { leftover.remove() } }
    }
    count += 1
    block.parentElement.replaceWith(holder)
  }
}
window.addEventListener('load', async () => {
  await drawDiagrams()
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

/**
 * The preview of a document: the HTML the engine exported, with the policy and the script added, and the bundled Mermaid
 * library when it is given, so diagrams are drawn.
 */
export function renderHtmlPreview (html: string, nonce: string, mermaid?: MermaidSource): string {
  const head = html.indexOf('<head>')
  const rule = policy(nonce, mermaid?.cspSource)
  const withPolicy = head === -1
    ? rule + html
    : `${html.slice(0, head + 6)}
${rule}${html.slice(head + 6)}`
  const library = mermaid
    ? `<script nonce="${nonce}" src="${mermaid.scriptUri}"></script>
`
    : ''
  // The last one, in case the document itself writes `</body>`.
  const body = withPolicy.lastIndexOf('</body>')
  const scripts = `${library}${script(nonce)}`

  return body === -1
    ? withPolicy + scripts
    : `${withPolicy.slice(0, body)}${scripts}
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
