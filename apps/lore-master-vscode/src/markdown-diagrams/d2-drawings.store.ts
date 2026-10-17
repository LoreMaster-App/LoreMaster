import { DIAGRAM_RENDER_METHOD, type DiagramRenderParams, type DiagramRenderResult } from '../engine-protocol'
import type { DiagramDrawings, Drawing, DrawingEngine } from './markdown-diagrams.contract'

const DRAWING: Drawing = { state: 'drawing' }

/** How many diagrams are remembered; the oldest go first. */
export const REMEMBERED_DRAWINGS = 200

/**
 * The D2 drawings of the preview. Markdown-it renders synchronously and the engine answers later, so a diagram not yet seen is
 * reported as being drawn, sent to the engine once, and `onDrawn` fires when the answer is in so the preview renders again and
 * finds it here.
 */
export class D2Drawings implements DiagramDrawings {
  private readonly drawings = new Map<string, Drawing>()

  constructor (
    private readonly engine: DrawingEngine,
    private readonly onDrawn: () => void,
    private readonly limit = REMEMBERED_DRAWINGS,
  ) {}

  private async draw (source: string): Promise<void> {
    const params: DiagramRenderParams = { language: 'd2', source }
    let outcome: Drawing
    try {
      const result = await this.engine.request<DiagramRenderResult>(DIAGRAM_RENDER_METHOD, params)
      outcome = { state: 'drawn', svg: result.svg }
    } catch (error) {
      outcome = { state: 'failed', message: error instanceof Error ? error.message : String(error) }
    }
    this.drawings.set(source, outcome)
    while (this.drawings.size > this.limit) {
      this.drawings.delete(this.drawings.keys().next().value as string)
    }
    this.onDrawn()
  }

  drawingOf (source: string): Drawing {
    const known = this.drawings.get(source)
    if (known) {
      return known
    }
    this.drawings.set(source, DRAWING)
    void this.draw(source)

    return DRAWING
  }
}
