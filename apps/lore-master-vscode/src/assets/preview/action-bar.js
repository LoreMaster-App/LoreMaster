// Runs inside VS Code's built-in Markdown preview (contributed as `markdown.previewScripts`).
// Drives the switches of LoreMaster's action bar. A switch adds a class to the page (`loremaster-off-<key>`), which the preview
// style (action-bar-preview.css) turns into hiding or undoing that addition. The state is kept in the page's storage, so it holds
// across renders and reloads of the preview; where storage is blocked the switches still work until the preview is closed.
(function () {
  'use strict'

  const STORAGE_KEY = 'loremaster.preview.off'
  let off = new Set()

  try {
    off = new Set(JSON.parse(window.localStorage.getItem(STORAGE_KEY) || '[]'))
  } catch (error) {
    off = new Set()
  }

  function save () {
    try { window.localStorage.setItem(STORAGE_KEY, JSON.stringify(Array.from(off))) } catch (error) { /* storage is blocked */ }
  }

  function apply () {
    for (const key of ['status', 'callouts', 'diagrams']) {
      document.documentElement.classList.toggle('loremaster-off-' + key, off.has(key))
    }
    for (const toggle of document.querySelectorAll('.loremaster-bar-toggle')) {
      toggle.setAttribute('aria-pressed', off.has(toggle.dataset.toggle) ? 'false' : 'true')
    }
  }

  document.addEventListener('click', event => {
    const toggle = event.target.closest && event.target.closest('.loremaster-bar-toggle')
    if (!toggle) { return }
    event.preventDefault()
    const key = toggle.dataset.toggle
    if (off.has(key)) { off.delete(key) } else { off.add(key) }
    save()
    apply()
  })

  // The preview replaces the page's content as the file is edited, and tells the scripts when it has.
  window.addEventListener('vscode.markdown.updateContent', apply)
  window.addEventListener('load', apply)
  document.addEventListener('DOMContentLoaded', apply)
  apply()
})()
