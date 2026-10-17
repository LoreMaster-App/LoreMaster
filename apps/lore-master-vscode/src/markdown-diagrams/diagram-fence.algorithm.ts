import type { Drawing, DiagramDrawings, MarkdownItLike } from './markdown-diagrams.contract'

const escapeHtml = (text: string): string => text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')

const languageOf = (info: string): string => info.trim().split(/\s+/, 1)[0].toLowerCase()

/** The markup of a D2 fence by how far its drawing has got. */
function d2Markup (drawing: Drawing, attributes: string): string {
  switch (drawing.state) {
    case 'drawn': {
      return `<div${attributes}>${drawing.svg}</div>\n`
    }
    case 'failed': {
      return `<div${attributes}><span class="loremaster-diagram-error">The D2 diagram could not be drawn: ${escapeHtml(drawing.message)}</span></div>\n`
    }
    case 'drawing': {
      return `<div${attributes}><span class="loremaster-diagram-pending">Drawing the D2 diagram...</span></div>\n`
    }
  }
}

/** Which fences the preview is taught. */
export interface DiagramFenceOptions {
  /** Draw ```mermaid too: false when VS Code draws Mermaid itself. */
  mermaid: boolean
}

/**
 * Teaches VS Code's own Markdown preview two fences: ```d2 is drawn by the engine (through `drawings`) and inlined as SVG, and
 * ```mermaid is left as source in a marked block for the preview script to draw in the page, unless VS Code draws Mermaid itself.
 * Every other fence is rendered as it was.
 */
export function addDiagramFences<Md extends MarkdownItLike> (md: Md, drawings: DiagramDrawings, fences: DiagramFenceOptions): Md {
  const fallback = md.renderer.rules.fence

  md.renderer.rules.fence = (tokens, index, options, env, self) => {
    const token = tokens[index]
    const language = languageOf(token.info)
    if (language !== 'd2' && !(language === 'mermaid' && fences.mermaid)) {
      return fallback ? fallback(tokens, index, options, env, self) : `<pre><code>${escapeHtml(token.content)}</code></pre>\n`
    }
    token.attrJoin('class', 'loremaster-diagram')
    token.attrJoin('class', `loremaster-${language}`)
    const attributes = self.renderAttrs(token)

    return language === 'mermaid'
      ? `<pre${attributes}>${escapeHtml(token.content)}</pre>\n`
      : d2Markup(drawings.drawingOf(token.content), attributes)
  }

  return md
}
