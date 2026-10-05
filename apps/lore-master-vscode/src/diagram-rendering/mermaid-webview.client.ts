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

/**
 * A {@link DiagramRenderer} backed by a webview that runs Mermaid (bundled at
 * `dist/assets/webview/mermaid.js`), so diagrams render with the same library a Markdown
 * preview uses. The panel is created on first use, kept for reuse (`retainContextWhenHidden`)
 * and disposed on deactivate. It opens beside the editor without taking focus.
 */
export function createMermaidRenderer (extensionUri: vscode.Uri): DiagramRenderer {
  let panel: vscode.WebviewPanel | undefined
  let sequence = 0
  const pending = new Map<number, Pending>()

  function ensurePanel (): vscode.Webview {
    if (panel) {
      return panel.webview
    }
    panel = vscode.window.createWebviewPanel(
      'loreMasterDiagrams',
      'LoreMaster diagrams',
      { viewColumn: vscode.ViewColumn.Beside, preserveFocus: true },
      { enableScripts: true, retainContextWhenHidden: true, localResourceRoots: [vscode.Uri.joinPath(extensionUri, 'dist', 'assets')] },
    )
    const mermaid = panel.webview.asWebviewUri(vscode.Uri.joinPath(extensionUri, 'dist', 'assets', 'webview', 'mermaid.min.js'))
    panel.webview.html = pageHtml(panel.webview, mermaid)
    panel.webview.onDidReceiveMessage((reply: RenderReply) => {
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
    panel.onDidDispose(() => {
      panel = undefined
      for (const waiting of pending.values()) {
        waiting.reject(new Error('the diagram renderer was closed'))
      }
      pending.clear()
    })

    return panel.webview
  }

  return {
    render (source: string): Promise<string> {
      const webview = ensurePanel()
      const id = ++sequence

      return new Promise<string>((resolve, reject) => {
        pending.set(id, { resolve, reject })
        void webview.postMessage({ id, source })
      })
    },
    dispose (): void {
      panel?.dispose()
      panel = undefined
    },
  }
}

function pageHtml (webview: vscode.Webview, mermaid: vscode.Uri): string {
  const nonce = nonceString()

  // Mermaid's own prebuilt bundle (global `mermaid`) plus a small glue script that renders
  // on request and posts the SVG back. Both run under a nonce; nothing else is allowed.
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src ${webview.cspSource} data:; script-src 'nonce-${nonce}' ${webview.cspSource}; style-src 'unsafe-inline' ${webview.cspSource};" />
</head>
<body>
  <script nonce="${nonce}" src="${mermaid.toString()}"></script>
  <script nonce="${nonce}">
    mermaid.initialize({ startOnLoad: false });
    const vscode = acquireVsCodeApi();
    window.addEventListener('message', async (event) => {
      const { id, source } = event.data;
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
