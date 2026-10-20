// Runs inside VS Code's built-in Markdown preview (contributed as `markdown.previewScripts`).
// Reading aids: a Copy button on code blocks, a link icon on headings, and a floating outline of the headings.
(function () {
  'use strict'

  function button (label, title, onClick) {
    const made = document.createElement('button')
    made.type = 'button'
    made.className = 'loremaster-reading-button'
    made.textContent = label
    made.title = title
    made.setAttribute('aria-label', title)
    made.addEventListener('click', event => { event.stopPropagation(); onClick(made) })
    return made
  }

  function copyText (text, made, idle) {
    const done = ok => {
      made.textContent = ok ? 'Copied' : 'Failed'
      setTimeout(() => { made.textContent = idle }, 1500)
    }
    if (!navigator.clipboard) { done(false); return }
    navigator.clipboard.writeText(text).then(() => done(true), () => done(false))
  }

  // Code blocks. Diagrams are LoreMaster's own blocks and have their own buttons.
  function decorateCode () {
    for (const block of document.querySelectorAll('pre')) {
      if (block.classList.contains('loremaster-diagram') || block.querySelector('.loremaster-code-tools')) { continue }
      const code = block.querySelector('code')
      if (!code) { continue }
      const tools = document.createElement('div')
      tools.className = 'loremaster-code-tools'
      tools.appendChild(button('Copy', 'Copy the code', made => copyText(code.textContent, made, 'Copy')))
      block.classList.add('loremaster-code')
      block.appendChild(tools)
    }
  }

  // Headings: a link to the heading, shown on hover.
  function headings () {
    return Array.from(document.querySelectorAll('h1[id], h2[id], h3[id], h4[id]'))
  }

  function decorateHeadings () {
    for (const heading of headings()) {
      if (heading.querySelector('.loremaster-anchor')) { continue }
      const link = document.createElement('a')
      link.className = 'loremaster-anchor'
      link.href = '#' + heading.id
      link.textContent = '#'
      link.title = 'Link to this heading'
      link.setAttribute('aria-label', 'Link to this heading')
      heading.appendChild(link)
    }
  }

  // Outline: a toggle at the top right that opens a list of the headings.
  let outline
  let open = false

  function outlineList () {
    const list = document.createElement('ul')
    list.className = 'loremaster-outline-list'
    for (const heading of headings()) {
      const item = document.createElement('li')
      item.className = 'loremaster-outline-' + heading.tagName.toLowerCase()
      const link = document.createElement('a')
      link.textContent = Array.from(heading.childNodes).filter(node => !(node.classList && node.classList.contains('loremaster-anchor'))).map(node => node.textContent).join('').trim()
      link.href = '#' + heading.id
      link.addEventListener('click', event => { event.preventDefault(); heading.scrollIntoView({ block: 'start' }) })
      item.appendChild(link)
      list.appendChild(item)
    }
    return list
  }

  function renderOutline () {
    if (!outline) {
      outline = document.createElement('nav')
      outline.className = 'loremaster-outline'
      outline.setAttribute('aria-label', 'Outline')
      document.body.appendChild(outline)
    }
    const toggle = button(open ? 'Outline ×' : 'Outline', 'Show or hide the outline', () => { open = !open; renderOutline() })
    toggle.classList.add('loremaster-outline-toggle')
    outline.replaceChildren(toggle)
    if (open && headings().length > 0) { outline.appendChild(outlineList()) }
    outline.hidden = headings().length === 0
  }

  let pending
  function update () {
    clearTimeout(pending)
    pending = setTimeout(() => { decorateCode(); decorateHeadings(); renderOutline() }, 50)
  }

  // The preview replaces the page's content as the file is edited, and tells the scripts when it has.
  window.addEventListener('vscode.markdown.updateContent', update)
  window.addEventListener('load', update)
  document.addEventListener('DOMContentLoaded', update)
})()
