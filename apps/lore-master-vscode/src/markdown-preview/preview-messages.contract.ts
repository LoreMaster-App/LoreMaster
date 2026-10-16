/** What the preview page tells the extension. */
export type PreviewToHost =
  | { type: 'ready' } |
  { type: 'scroll', fraction: number } |
  { type: 'open', href: string }

/** What the extension tells the preview page. */
export interface HostToPreview {
  type:     'scroll'
  /** How far down the document the editor is, from 0 (top) to 1 (bottom). */
  fraction: number
}
