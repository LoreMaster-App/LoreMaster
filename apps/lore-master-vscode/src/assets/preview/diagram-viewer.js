// Runs inside VS Code's built-in Markdown preview (contributed as `markdown.previewScripts`).
// Gives every drawn LoreMaster diagram (Mermaid and D2) two hover buttons - enlarge and copy SVG - and an enlarged view
// with wheel zoom, drag to pan, double-click to reset and Escape to close.
(function () {
  'use strict'

  const MIN_SCALE = 0.2
  const MAX_SCALE = 20

  function button (label, title, onClick) {
    const made = document.createElement('button')
    made.type = 'button'
    made.className = 'loremaster-viewer-button'
    made.textContent = label
    made.title = title
    made.setAttribute('aria-label', title)
    made.addEventListener('click', event => { event.stopPropagation(); onClick(made) })
    return made
  }

  function copySvg (svg, made) {
    const text = new XMLSerializer().serializeToString(svg)
    const done = ok => {
      made.textContent = ok ? 'Copied' : 'Failed'
      setTimeout(() => { made.textContent = 'Copy SVG' }, 1500)
    }
    if (!navigator.clipboard) { done(false); return }
    navigator.clipboard.writeText(text).then(() => done(true), () => done(false))
  }

  function enlarge (svg, light) {
    const overlay = document.createElement('div')
    overlay.className = 'loremaster-viewer-overlay'
    const stage = document.createElement('div')
    stage.className = light ? 'loremaster-viewer-stage loremaster-viewer-light' : 'loremaster-viewer-stage'
    const copy = svg.cloneNode(true)
    copy.removeAttribute('width')
    copy.removeAttribute('height')
    copy.style.maxWidth = '100%'
    copy.style.maxHeight = '100%'
    stage.appendChild(copy)

    let scale = 1
    let x = 0
    let y = 0
    const apply = () => { stage.style.transform = 'translate(' + x + 'px, ' + y + 'px) scale(' + scale + ')' }
    const zoom = factor => { scale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale * factor)); apply() }
    const reset = () => { scale = 1; x = 0; y = 0; apply() }
    const close = () => { document.removeEventListener('keydown', onKey, true); overlay.remove() }
    function onKey (event) {
      if (event.key === 'Escape') { close() } else if (event.key === '+' || event.key === '=') { zoom(1.25) } else if (event.key === '-') { zoom(0.8) } else if (event.key === '0') { reset() } else { return }
      event.preventDefault()
      event.stopPropagation()
    }

    const bar = document.createElement('div')
    bar.className = 'loremaster-viewer-bar'
    bar.append(
      button('+', 'Zoom in', () => zoom(1.25)),
      button('-', 'Zoom out', () => zoom(0.8)),
      button('Reset', 'Reset the view', reset),
      button('Close', 'Close', close),
    )

    overlay.addEventListener('wheel', event => { event.preventDefault(); zoom(Math.exp(-event.deltaY * 0.002)) }, { passive: false })
    overlay.addEventListener('dblclick', reset)
    overlay.addEventListener('mousedown', event => {
      if (event.target.closest('.loremaster-viewer-bar')) { return }
      const startX = event.clientX - x
      const startY = event.clientY - y
      let moved = false
      const move = other => { moved = true; x = other.clientX - startX; y = other.clientY - startY; apply() }
      const up = () => {
        window.removeEventListener('mousemove', move)
        window.removeEventListener('mouseup', up)
        if (!moved && event.target === overlay) { close() }
      }
      window.addEventListener('mousemove', move)
      window.addEventListener('mouseup', up)
      event.preventDefault()
    })
    document.addEventListener('keydown', onKey, true)
    overlay.append(stage, bar)
    document.body.appendChild(overlay)
  }

  function decorate (block) {
    const svg = block.querySelector('svg')
    if (!svg || block.querySelector('.loremaster-viewer-tools')) { return }
    const tools = document.createElement('div')
    tools.className = 'loremaster-viewer-tools'
    tools.append(
      button('Enlarge', 'Open the diagram enlarged', () => enlarge(svg, block.classList.contains('loremaster-d2'))),
      button('Copy SVG', 'Copy the diagram as SVG', made => copySvg(svg, made)),
    )
    block.appendChild(tools)
  }

  function decorateAll () {
    for (const block of document.querySelectorAll('.loremaster-diagram')) { decorate(block) }
  }

  // Diagrams are drawn, redrawn on a theme change, and replaced as the file is edited; decorate whatever shows up.
  new MutationObserver(decorateAll).observe(document.body, { childList: true, subtree: true })
  window.addEventListener('vscode.markdown.updateContent', decorateAll)
  window.addEventListener('load', decorateAll)
  document.addEventListener('DOMContentLoaded', decorateAll)
})()
