import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { exportDocumentAsText } from '../export-markdown'
import type { HostToPreview, PreviewToHost } from './preview-messages.contract'
import { classifyLink } from './preview-link.policy'
import { fractionOfLine, lineOfFraction } from './preview-scroll.algorithm'
import { renderHtmlPreview, renderTextPage } from './render-preview-page.mapper'

/** The id of the preview's webview panel. */
export const PREVIEW_VIEW_TYPE = 'loreMasterMarkdownPreview'

/** The format that shows the page as it will read, as opposed to the text of another format. */
export const RENDERED = 'html'

const REFRESH_DELAY = 250
/** How long after the preview moves the editor the editor's own scrolling is not passed back, so the two do not chase each other. */
const ECHO_WINDOW = 200

const settings = (): vscode.WorkspaceConfiguration => vscode.workspace.getConfiguration('loreMaster.markdown.preview')

function nonce (): string {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let value = ''
  for (let index = 0; index < 32; index += 1) {
    value += alphabet.charAt(Math.floor(Math.random() * alphabet.length))
  }

  return value
}

/**
 * LoreMaster's Markdown preview: one panel that shows a document the way the engine exports it, follows the editor, updates as
 * the text changes (unsaved changes included), scrolls with the editor and back, and can show the text of another format
 * instead, so the author sees how a page will convert before syncing or pasting it.
 */
export class MarkdownPreview implements vscode.Disposable {
  private panel:    vscode.WebviewPanel | undefined
  private document: vscode.TextDocument | undefined
  private format = RENDERED
  private formatLabel = 'Rendered'
  private timer:    ReturnType<typeof setTimeout> | undefined
  private quietUntil = 0
  private generation = 0

  constructor (private readonly engine: EngineClient, private readonly extensionUri?: vscode.Uri) {}

  private title (): string {
    const name = this.document ? this.document.uri.path.split('/').pop() ?? '' : ''

    return this.format === RENDERED ? `Preview ${name}` : `Preview ${name} as ${this.formatLabel}`
  }

  /** The bundled Mermaid library as the panel can load it, when the extension knows where it is. */
  private mermaidIn (panel: vscode.WebviewPanel): { scriptUri: string, cspSource: string } | undefined {
    if (!this.extensionUri) {
      return undefined
    }
    const script = vscode.Uri.joinPath(this.extensionUri, 'dist', 'assets', 'webview', 'mermaid.min.js')

    return { scriptUri: panel.webview.asWebviewUri(script).toString(), cspSource: panel.webview.cspSource }
  }

  private schedule (): void {
    clearTimeout(this.timer)
    this.timer = setTimeout(() => { void this.refresh() }, REFRESH_DELAY)
  }

  /** Asks the engine for the document in the chosen format and puts it in the panel. A slow answer never overwrites a newer one. */
  private async refresh (): Promise<void> {
    const { panel, document } = this
    if (!panel || !document) {
      return
    }
    this.generation += 1
    const mine = this.generation
    try {
      const rendered = this.format === RENDERED
      const text = await exportDocumentAsText(this.engine, { path: document.uri.fsPath, content: document.getText(), format: this.format, preview: rendered })
      if (mine === this.generation && this.panel === panel) {
        panel.webview.html = rendered ? renderHtmlPreview(text, nonce(), this.mermaidIn(panel)) : renderTextPage(text, this.formatLabel, nonce())
      }
    } catch (error) {
      if (mine === this.generation && this.panel === panel) {
        panel.webview.html = renderTextPage(error instanceof Error ? error.message : String(error), 'The preview could not be made', nonce())
      }
    }
  }

  private receive (message: PreviewToHost): void {
    switch (message.type) {
      case 'ready': {
        this.syncFromEditor()
        break
      }
      case 'scroll': {
        this.scrollEditor(message.fraction)
        break
      }
      case 'open': {
        void this.open(message.href)
        break
      }
    }
  }

  /** When the page has loaded, put it where the editor is. */
  private syncFromEditor (): void {
    const editor = vscode.window.visibleTextEditors.find(each => each.document.uri.toString() === this.document?.uri.toString())
    if (editor && settings().get<boolean>('scrollSync', true) && this.format === RENDERED) {
      this.editorScrolled(editor)
    }
  }

  private scrollEditor (fraction: number): void {
    const editor = vscode.window.visibleTextEditors.find(each => each.document.uri.toString() === this.document?.uri.toString())
    if (!editor || !settings().get<boolean>('scrollSync', true)) {
      return
    }
    this.quietUntil = Date.now() + ECHO_WINDOW
    const line = lineOfFraction(fraction, editor.document.lineCount)
    editor.revealRange(new vscode.Range(line, 0, line, 0), vscode.TextEditorRevealType.AtTop)
  }

  private async open (href: string): Promise<void> {
    if (!this.document) {
      return
    }
    const action = classifyLink(href, this.document.uri.fsPath)
    if (action.kind === 'external') {
      await vscode.env.openExternal(vscode.Uri.parse(action.uri))
    } else if (action.kind === 'markdown') {
      try {
        const target = await vscode.workspace.openTextDocument(vscode.Uri.file(action.file))
        await vscode.window.showTextDocument(target, { viewColumn: vscode.ViewColumn.One, preview: false })
      } catch {
        await vscode.window.showWarningMessage(`LoreMaster: ${action.file} could not be opened.`)
      }
    }
  }

  /** Shows the preview of a document beside the editor, opening the panel or bringing it forward. */
  show (document: vscode.TextDocument): void {
    this.document = document
    if (this.panel) {
      this.panel.title = this.title()
      this.panel.reveal(vscode.ViewColumn.Beside, true)
    } else {
      this.panel = vscode.window.createWebviewPanel(PREVIEW_VIEW_TYPE, this.title(), { viewColumn: vscode.ViewColumn.Beside, preserveFocus: true }, { enableScripts: true, retainContextWhenHidden: true, localResourceRoots: this.extensionUri ? [vscode.Uri.joinPath(this.extensionUri, 'dist', 'assets')] : [] })
      this.panel.onDidDispose(() => { this.panel = undefined })
      this.panel.webview.onDidReceiveMessage((message: PreviewToHost) => this.receive(message))
    }
    void this.refresh()
  }

  /** Shows the text of another format instead of the rendered page, or goes back to the rendered page. */
  showAs (format: string, label: string): void {
    this.format = format
    this.formatLabel = label
    if (this.panel && this.document) {
      this.panel.title = this.title()
      void this.refresh()
    }
  }

  get isOpen (): boolean {
    return this.panel !== undefined
  }

  /** The text of the document changed: show it again once the typing pauses. */
  documentChanged (document: vscode.TextDocument): void {
    if (this.panel && this.document?.uri.toString() === document.uri.toString()) {
      this.schedule()
    }
  }

  /** The editor in front changed: follow it, when it is another Markdown file and the setting says to. */
  editorChanged (editor: vscode.TextEditor | undefined): void {
    if (!this.panel || editor?.document.languageId !== 'markdown' || !settings().get<boolean>('followActiveEditor', true)) {
      return
    }
    if (this.document?.uri.toString() !== editor.document.uri.toString()) {
      this.document = editor.document
      this.panel.title = this.title()
      void this.refresh()
    }
  }

  /** The editor scrolled: scroll the preview to the same place. */
  editorScrolled (editor: vscode.TextEditor): void {
    if (!this.panel || this.format !== RENDERED || !settings().get<boolean>('scrollSync', true) || Date.now() < this.quietUntil) {
      return
    }
    if (editor.document.uri.toString() !== this.document?.uri.toString() || editor.visibleRanges.length === 0) {
      return
    }
    const message: HostToPreview = { type: 'scroll', fraction: fractionOfLine(editor.visibleRanges[0].start.line, editor.document.lineCount) }
    void this.panel.webview.postMessage(message)
  }

  dispose (): void {
    clearTimeout(this.timer)
    this.panel?.dispose()
  }
}
