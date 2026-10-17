import { addDiagramFences } from './diagram-fence.algorithm'
import type { DiagramDrawings, Drawing, FenceRule, FenceToken, MarkdownItLike } from './markdown-diagrams.contract'

function token (info: string, content: string): FenceToken & { classes: string[] } {
  const classes: string[] = []

  return { info, content, classes, attrJoin: (_name, value) => { classes.push(value) } }
}

const self = { renderAttrs: (made: FenceToken) => ` class="${(made as FenceToken & { classes: string[] }).classes.join(' ')}" data-line="3"` }

function markdown (): { md: MarkdownItLike, fallbackCalls: string[] } {
  const fallbackCalls: string[] = []
  const fallback: FenceRule = tokens => {
    fallbackCalls.push(tokens[0].info)

    return '<pre>plain</pre>'
  }

  return { md: { renderer: { rules: { fence: fallback } } }, fallbackCalls }
}

const drawings = (drawing: Drawing): DiagramDrawings & { asked: string[] } => {
  const asked: string[] = []

  return {
    asked,
    drawingOf: source => {
      asked.push(source)

      return drawing
    },
  }
}

function render (md: MarkdownItLike, made: FenceToken): string {
  return (md.renderer.rules.fence as FenceRule)([made], 0, {}, {}, self)
}

describe('addDiagramFences', () => {
  it('leaves a mermaid fence as escaped source in a marked block for the page to draw', () => {
    const { md } = markdown()
    addDiagramFences(md, drawings({ state: 'drawing' }), { mermaid: true })

    const html = render(md, token('mermaid', 'graph TD\n  A-->B & C'))

    expect(html).toBe('<pre class="loremaster-diagram loremaster-mermaid" data-line="3">graph TD\n  A--&gt;B &amp; C</pre>\n')
  })

  it('does not ask the engine for a mermaid diagram', () => {
    const { md } = markdown()
    const known = drawings({ state: 'drawing' })
    addDiagramFences(md, known, { mermaid: true })

    render(md, token('mermaid', 'graph TD'))

    expect(known.asked).toEqual([])
  })

  it('inlines a drawn D2 diagram', () => {
    const { md } = markdown()
    const known = drawings({ state: 'drawn', svg: '<svg>x</svg>' })
    addDiagramFences(md, known, { mermaid: true })

    const html = render(md, token('d2', 'a -> b'))

    expect(html).toBe('<div class="loremaster-diagram loremaster-d2" data-line="3"><svg>x</svg></div>\n')
    expect(known.asked).toEqual(['a -> b'])
  })

  it('says a D2 diagram is being drawn', () => {
    const { md } = markdown()
    addDiagramFences(md, drawings({ state: 'drawing' }), { mermaid: true })

    expect(render(md, token('d2', 'a -> b'))).toContain('Drawing the D2 diagram...')
  })

  it('shows why a D2 diagram failed, escaped', () => {
    const { md } = markdown()
    addDiagramFences(md, drawings({ state: 'failed', message: 'bad <token>' }), { mermaid: true })

    expect(render(md, token('d2', 'a -> {'))).toContain('The D2 diagram could not be drawn: bad &lt;token&gt;')
  })

  it('takes the language from the first word of the info string, in any case', () => {
    const { md } = markdown()
    addDiagramFences(md, drawings({ state: 'drawn', svg: '<svg/>' }), { mermaid: true })

    expect(render(md, token('  D2 title=x', 'a'))).toContain('loremaster-d2')
  })

  it('leaves mermaid to VS Code when it draws Mermaid itself, and still draws d2', () => {
    const { md, fallbackCalls } = markdown()
    addDiagramFences(md, drawings({ state: 'drawn', svg: '<svg/>' }), { mermaid: false })

    expect(render(md, token('mermaid', 'graph TD'))).toBe('<pre>plain</pre>')
    expect(fallbackCalls).toEqual(['mermaid'])
    expect(render(md, token('d2', 'a'))).toContain('loremaster-d2')
  })

  it('hands every other fence to the rule it replaced', () => {
    const { md, fallbackCalls } = markdown()
    addDiagramFences(md, drawings({ state: 'drawing' }), { mermaid: true })

    expect(render(md, token('ts', 'let a'))).toBe('<pre>plain</pre>')
    expect(fallbackCalls).toEqual(['ts'])
  })
})
