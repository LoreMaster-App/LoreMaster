import { HOST_RENDER_DIAGRAM_METHOD, type RenderDiagramParams, type RenderDiagramResult } from '../engine-protocol'
import type { EngineClient } from '../engine-process'

/** Draws a diagram's source into an SVG, the same way the editor's Markdown preview does.
 *  The webview-backed implementation is `createMermaidRenderer`. */
export interface DiagramRenderer {
  render (source: string): Promise<string>
  dispose (): void
}

/**
 * Answers the engine's `host/renderDiagram` requests with the renderer, so a sync's
 * `image` Mermaid mode gets the drawing the author saw. Only Mermaid is supported; any
 * other language is refused, which the engine reports. Returns the subscription to dispose
 * on deactivate.
 */
export function answerRenderDiagrams (deps: { engine: EngineClient; renderer: DiagramRenderer }): { dispose (): void } {
  return deps.engine.onRequest(HOST_RENDER_DIAGRAM_METHOD, async params => {
    const { language, source } = params as RenderDiagramParams
    if (language !== 'mermaid') {
      throw new Error(`LoreMaster cannot render a ${language} diagram`)
    }

    return { svg: await deps.renderer.render(source) } satisfies RenderDiagramResult
  })
}
