import * as vscode from 'vscode'
import type { DiagramRenderer } from './diagram-render.handler'

interface Pending {
  resolve: (svg: string) => void
  reject:  (error: Error) => void
}

interface RenderReply {
  id:     number
  svg?:   string
  error?: string
}

/** The part of a webview panel the renderer uses, so a test can stand in for the editor's. */
export interface DiagramPanel {
  readonly webview: {
    html: string
    postMessage (message: unknown): PromiseLike<boolean>
    onDidReceiveMessage (listener: (message: RenderReply) => unknown): { dispose (): unknown }
  }
  onDidDispose (listener: () => unknown): { dispose (): unknown }
  dispose (): void
}

export interface MermaidRendererOptions {
  /** How long one diagram may take before it is given up on and the renderer restarted. */
  budgetMs?:  number
  /** Opens the panel; the editor's webview panel unless a test supplies one. */
  openPanel?: (extensionUri: vscode.Uri) => { panel: DiagramPanel, mermaidUri: string, cspSource: string }
}

/** Under the engine's 30 s wait per diagram, so a stuck one is named before the engine gives up. */
export const DEFAULT_RENDER_BUDGET_MS = 20_000

function openEditorPanel (extensionUri: vscode.Uri): { panel: DiagramPanel, mermaidUri: string, cspSource: string } {
  const panel = vscode.window.createWebviewPanel(
    'loreMasterDiagrams',
    'LoreMaster diagrams',
    { viewColumn: vscode.ViewColumn.Beside, preserveFocus: true },
    { enableScripts: true, retainContextWhenHidden: true, localResourceRoots: [vscode.Uri.joinPath(extensionUri, 'dist', 'assets')] },
  )
  const mermaid = panel.webview.asWebviewUri(vscode.Uri.joinPath(extensionUri, 'dist', 'assets', 'webview', 'mermaid.min.js'))

  return { panel, mermaidUri: mermaid.toString(), cspSource: panel.webview.cspSource }
}

/**
 * A {@link DiagramRenderer} backed by a webview that runs Mermaid (bundled at
 * `dist/assets/webview/mermaid.min.js`), so diagrams render with the same library a Markdown
 * preview uses. The panel is created on first use, kept for reuse (`retainContextWhenHidden`)
 * and disposed on deactivate. It opens beside the editor without taking focus.
 *
 * Mermaid renders one diagram at a time, so one that never finishes would hold every later
 * one behind it. Each diagram therefore has a budget: when it is spent, that diagram fails by
 * name and the panel is replaced, so the next one starts on a fresh Mermaid.
 */
export function createMermaidRenderer (extensionUri: vscode.Uri, options: MermaidRendererOptions = {}): DiagramRenderer {
  const budgetMs = options.budgetMs ?? DEFAULT_RENDER_BUDGET_MS
  const open = options.openPanel ?? openEditorPanel
  let panel: DiagramPanel | undefined
  let sequence = 0
  const pending = new Map<number, Pending>()

  function ensurePanel (): DiagramPanel {
    if (panel) {
      return panel
    }
    const opened = open(extensionUri)
    const created = opened.panel
    panel = created
    created.webview.html = pageHtml(opened.cspSource, opened.mermaidUri)
    created.webview.onDidReceiveMessage((reply: RenderReply) => {
      const waiting = pending.get(reply.id)
      if (!waiting) {
        return
      }
      pending.delete(reply.id)
      if (reply.error === undefined) {
        waiting.resolve(reply.svg ?? '')
      } else {
        waiting.reject(new Error(reply.error))
      }
    })
    created.onDidDispose(() => {
      if (panel === created) {
        panel = undefined
      }
      for (const waiting of pending.values()) {
        waiting.reject(new Error('the diagram renderer was closed'))
      }
      pending.clear()
    })

    return created
  }

  return {
    render (source: string): Promise<string> {
      const target = ensurePanel()
      const id = ++sequence

      return new Promise<string>((resolve, reject) => {
        const timer = setTimeout(() => {
          if (!pending.delete(id)) {
            return
          }
          // The stuck diagram is already forgotten, so disposing rejects nobody else with it.
          if (panel === target) {
            panel = undefined
          }
          target.dispose()
          reject(new Error(`the diagram did not finish drawing within ${Math.round(budgetMs / 1000)}s; the renderer was restarted so the next diagram is not held up`))
        }, budgetMs)
        pending.set(id, {
          resolve: svg => { clearTimeout(timer); resolve(svg) },
          reject:  error => { clearTimeout(timer); reject(error) },
        })
        void target.webview.postMessage({ id, source })
      })
    },
    dispose (): void {
      panel?.dispose()
      panel = undefined
    },
  }
}

function pageHtml (cspSource: string, mermaidUri: string): string {
  const nonce = nonceString()

  // Mermaid's own prebuilt bundle (global `mermaid`) plus a small glue script that renders
  // on request and posts the SVG back. Both run under a nonce; nothing else is allowed. The
  // listener is registered before Mermaid is touched, so a bundle that did not load answers
  // every request with that fact instead of leaving the engine waiting.
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src ${cspSource} data:; script-src 'nonce-${nonce}' ${cspSource}; style-src 'unsafe-inline' ${cspSource};" />
</head>
<body>
  <script nonce="${nonce}" src="${mermaidUri}"></script>
  <script nonce="${nonce}">
    const vscode = acquireVsCodeApi();
    let startupError;
    try {
      mermaid.initialize({ startOnLoad: false });
    } catch (error) {
      startupError = 'the Mermaid library did not load in the webview: ' + ((error && error.message) ? error.message : String(error));
    }
    window.addEventListener('message', async (event) => {
      const { id, source } = event.data;
      if (startupError) {
        vscode.postMessage({ id: id, error: startupError });
        return;
      }
      try {
        const { svg } = await mermaid.render('diagram-' + id, source);
        vscode.postMessage({ id: id, svg: svg });
      } catch (error) {
        vscode.postMessage({ id: id, error: (error && error.message) ? error.message : String(error) });
      }
    });
  </script>
</body>
</html>`
}

function nonceString (): string {
  let nonce = ''
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  for (let index = 0; index < 32; index += 1) {
    nonce += alphabet.charAt(Math.floor(Math.random() * alphabet.length))
  }

  return nonce
}
