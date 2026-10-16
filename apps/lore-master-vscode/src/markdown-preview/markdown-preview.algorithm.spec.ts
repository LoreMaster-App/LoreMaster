import * as path from 'node:path'
import { classifyLink } from './preview-link.policy'
import { fractionOfLine, lineOfFraction } from './preview-scroll.algorithm'
import { renderHtmlPreview, renderTextPage } from './render-preview-page.mapper'

describe('scroll positions', () => {
  it('maps a line to how far down the document it is and back', () => {
    expect(fractionOfLine(0, 101)).toBe(0)
    expect(fractionOfLine(50, 101)).toBe(0.5)
    expect(fractionOfLine(100, 101)).toBe(1)
    expect(lineOfFraction(0.5, 101)).toBe(50)
    expect(lineOfFraction(1, 101)).toBe(100)
  })

  it('stays inside the document, and copes with a one-line one', () => {
    expect(fractionOfLine(500, 10)).toBe(1)
    expect(lineOfFraction(-1, 10)).toBe(0)
    expect(lineOfFraction(7, 10)).toBe(9)
    expect(fractionOfLine(0, 1)).toBe(0)
    expect(lineOfFraction(0.5, 1)).toBe(0)
  })
})

describe('classifyLink', () => {
  const document = path.resolve('/work/docs/guide.md')

  it('opens web and mail addresses outside', () => {
    expect(classifyLink('https://example.com/a?b=1#c', document)).toEqual({ kind: 'external', uri: 'https://example.com/a?b=1#c' })
    expect(classifyLink('mailto:a@b.c', document)).toEqual({ kind: 'external', uri: 'mailto:a@b.c' })
  })

  it('opens a link to a Markdown file relative to the document, with its heading', () => {
    expect(classifyLink('../README.md#install', document)).toEqual({ kind: 'markdown', file: path.resolve('/work/README.md'), fragment: 'install' })
    expect(classifyLink('sub%20dir/a.markdown', document)).toEqual({ kind: 'markdown', file: path.resolve('/work/docs/sub dir/a.markdown'), fragment: '' })
  })

  it('does nothing for anything else, so the preview cannot open arbitrary files', () => {
    for (const href of ['image.png', 'script.js', 'file:///etc/passwd', 'vscode://x/y', '//host/share/a.md', 'javascript:alert(1)', '%E0%A4%A.md']) {
      expect(classifyLink(href, document)).toEqual({ kind: 'ignore' })
    }
  })
})

describe('the preview page', () => {
  const exported = '<!doctype html><html><head>\n<meta charset="utf-8">\n<title>T</title>\n</head>\n<body>\n<main><p>&lt;/body&gt; text</p></main>\n</body>\n</html>\n'

  it('adds a policy that lets only its own script run, and the script', () => {
    const page = renderHtmlPreview(exported, 'abc123')

    expect(page).toContain('script-src \'nonce-abc123\'')
    expect(page).toContain('default-src \'none\'')
    expect(page).toContain('<script nonce="abc123">')
    expect(page.indexOf('Content-Security-Policy')).toBeLessThan(page.indexOf('<meta charset'))
    expect(page.match(/<script/g)).toHaveLength(1)
  })

  it('loads the bundled Mermaid library and lets the page use it, only when it is given', () => {
    const plain = renderHtmlPreview(exported, 'n')
    const drawn = renderHtmlPreview(exported, 'n', { scriptUri: 'vscode-resource://x/mermaid.min.js', cspSource: 'vscode-resource:' })

    expect(plain).not.toContain('mermaid.min.js')
    expect(drawn).toContain('<script nonce="n" src="vscode-resource://x/mermaid.min.js"></script>')
    expect(drawn).toContain('script-src \'nonce-n\' vscode-resource:')
    expect(drawn.indexOf('mermaid.min.js')).toBeLessThan(drawn.indexOf('drawDiagrams'))
    expect(drawn).toContain('securityLevel: \'strict\'')
  })

  it('puts the script before the last closing body tag', () => {
    const page = renderHtmlPreview('<head></head><body><p>a</p></body>', 'n')

    expect(page.indexOf('</script>')).toBeLessThan(page.lastIndexOf('</body>'))
    expect(page.endsWith('</body>')).toBe(true)
  })

  it('shows text from another format escaped, so it cannot run', () => {
    const page = renderTextPage('<script>alert(1)</script> & more', 'Jira wiki', 'n')

    expect(page).toContain('&lt;script&gt;alert(1)&lt;/script&gt; &amp; more')
    expect(page).not.toContain('<script>alert')
    expect(page).toContain('Jira wiki')
  })
})
