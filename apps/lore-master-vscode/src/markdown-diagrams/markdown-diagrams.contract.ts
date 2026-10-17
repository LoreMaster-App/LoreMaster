/** The part of a markdown-it token the diagram fences read and style. */
export interface FenceToken {
  info:    string
  content: string
  attrJoin (name: string, value: string): void
}

/** The part of markdown-it's renderer the fence rule needs. */
export interface FenceRenderer {
  renderAttrs (token: FenceToken): string
}

export type FenceRule = (tokens: FenceToken[], index: number, options: unknown, env: unknown, self: FenceRenderer) => string

/** The part of the markdown-it instance VS Code hands to an extension. */
export interface MarkdownItLike {
  renderer: { rules: { fence?: FenceRule } }
}

/** What VS Code's Markdown extension calls on an extension that contributes `markdown.markdownItPlugins`. */
export interface MarkdownItExtension {
  extendMarkdownIt<Md extends MarkdownItLike> (md: Md): Md
}

/** Where a D2 diagram stands: being drawn by the engine, drawn, or refused with the reason. */
export type Drawing =
  | { state: 'drawing' } |
  { state: 'drawn', svg: string } |
  { state: 'failed', message: string }

/** The drawings the preview's fence rule asks for, one per D2 source. */
export interface DiagramDrawings {
  /** What is known of this diagram now; asking for one not yet seen starts drawing it. */
  drawingOf (source: string): Drawing
}

/** The one thing drawing needs from the engine client: to make a request. */
export interface DrawingEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}
