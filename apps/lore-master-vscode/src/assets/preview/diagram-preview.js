// Runs inside VS Code's built-in Markdown preview (contributed as `markdown.previewScripts`).
// The extension's markdown-it plugin leaves each ```mermaid fence as <pre class="loremaster-mermaid"> holding the source,
// but only when this VS Code does not draw Mermaid itself; this draws it in place, loading the Mermaid library next to this
// file the first time one is needed. D2 arrives already drawn as SVG, so it needs nothing here.
(function () {
  'use strict'

  const here = document.currentScript ? document.currentScript.src : ''
  let library

  const drawn = new Map()
  let counter = 0
  let queue = Promise.resolve()

  function dark () {
    const classes = document.body.classList
    return classes.contains('vscode-dark') || (classes.contains('vscode-high-contrast') && !classes.contains('vscode-high-contrast-light'))
  }

  function fill (block, content) {
    const holder = document.createElement('div')
    holder.className = 'loremaster-diagram-drawing'
    if (typeof content === 'string') {
      holder.innerHTML = content
    } else {
      holder.appendChild(content)
    }
    block.replaceChildren(holder)
  }

  function failure (message) {
    const note = document.createElement('span')
    note.className = 'loremaster-diagram-error'
    note.textContent = 'The Mermaid diagram could not be drawn: ' + message
    return note
  }

  async function drawBlock (block) {
    // The source is kept on the block: the page is rebuilt on a theme change and redrawn from it.
    const source = block.dataset.source !== undefined ? block.dataset.source : block.textContent
    block.dataset.source = source
    const theme = dark() ? 'dark' : 'default'
    const key = theme + '\n' + source
    if (drawn.has(key)) {
      fill(block, drawn.get(key))
      block.dataset.drawnTheme = theme
      return
    }
    const id = 'loremaster-mermaid-' + counter++
    try {
      const { svg } = await mermaid.render(id, source)
      drawn.set(key, svg)
      fill(block, svg)
    } catch (error) {
      fill(block, failure(error && error.message ? error.message : String(error)))
      // Mermaid leaves its own error picture in the page when a diagram fails; the note above replaces it.
      for (const leftover of [document.getElementById(id), document.getElementById('d' + id)]) {
        if (leftover) { leftover.remove() }
      }
    }
    block.dataset.drawnTheme = theme
  }

  function loadLibrary () {
    if (typeof mermaid !== 'undefined') { return Promise.resolve() }
    if (!library) {
      library = new Promise((resolve, reject) => {
        const script = document.createElement('script')
        script.src = here.replace(/preview\/diagram-preview\.js(\?.*)?$/, 'webview/mermaid.min.js')
        script.onload = resolve
        script.onerror = () => { library = undefined; reject(new Error('the Mermaid library could not be loaded')) }
        document.head.appendChild(script)
      })
    }
    return library
  }

  function drawAll () {
    const theme = dark() ? 'dark' : 'default'
    const blocks = Array.from(document.querySelectorAll('pre.loremaster-mermaid')).filter(block => block.dataset.drawnTheme !== theme)
    if (blocks.length === 0) { return queue }
    queue = queue.then(async () => {
      try {
        await loadLibrary()
      } catch (error) {
        for (const block of blocks) { fill(block, failure(error.message)) }
        return
      }
      mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme })
      for (const block of blocks) { await drawBlock(block) }
    })
    return queue
  }

  // The preview replaces the page's content as the file is edited, and tells the scripts when it has.
  window.addEventListener('vscode.markdown.updateContent', drawAll)
  window.addEventListener('load', drawAll)
  document.addEventListener('DOMContentLoaded', drawAll)
  // A theme change swaps the body's classes without reloading the page.
  new MutationObserver(drawAll).observe(document.body, { attributes: true, attributeFilter: ['class'] })
})()
