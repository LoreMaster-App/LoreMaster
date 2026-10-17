import * as vscode from 'vscode'
import { D2Drawings } from './d2-drawings.store'
import { addDiagramFences } from './diagram-fence.algorithm'
import type { DrawingEngine, MarkdownItExtension } from './markdown-diagrams.contract'

/** VS Code's own command that renders the open Markdown previews again. */
const REFRESH_PREVIEW_COMMAND = 'markdown.preview.refresh'

/** VS Code's own Mermaid support for the Markdown preview, present in recent versions. */
const BUILT_IN_MERMAID = 'vscode.mermaid-markdown-features'

/** Diagrams that finish together refresh the preview once. */
const REFRESH_AFTER_MS = 80

/**
 * What the extension hands back from `activate` so VS Code's built-in Markdown preview draws Mermaid and D2 (the
 * `markdown.markdownItPlugins` contribution). The page-side half is the preview script and style in `assets/preview`.
 */
export function registerMarkdownDiagrams (deps: { engine: DrawingEngine }): MarkdownItExtension & vscode.Disposable {
  let pending: ReturnType<typeof setTimeout> | undefined
  const drawings = new D2Drawings(deps.engine, () => {
    clearTimeout(pending)
    pending = setTimeout(() => { void vscode.commands.executeCommand(REFRESH_PREVIEW_COMMAND).then(undefined, () => {}) }, REFRESH_AFTER_MS)
  })

  return {
    extendMarkdownIt: md => addDiagramFences(md, drawings, { mermaid: vscode.extensions.getExtension(BUILT_IN_MERMAID) === undefined }),
    dispose:          () => { clearTimeout(pending) },
  }
}
