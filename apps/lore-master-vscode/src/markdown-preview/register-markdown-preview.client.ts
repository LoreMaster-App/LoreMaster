import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { MarkdownPreview } from './markdown-preview.client'
import { OPEN_PREVIEW_COMMAND, openPreview, PREVIEW_AS_COMMAND, previewAs } from './open-markdown-preview.handler'

/** Registers the preview commands and keeps the preview in step with the editor. */
export function registerMarkdownPreview (deps: { engine: EngineClient }): vscode.Disposable {
  const preview = new MarkdownPreview(deps.engine)

  return vscode.Disposable.from(
    preview,
    vscode.commands.registerCommand(OPEN_PREVIEW_COMMAND, () => openPreview(preview)),
    vscode.commands.registerCommand(PREVIEW_AS_COMMAND, () => previewAs(preview, deps.engine)),
    vscode.workspace.onDidChangeTextDocument(event => { preview.documentChanged(event.document) }),
    vscode.window.onDidChangeActiveTextEditor(editor => { preview.editorChanged(editor) }),
    vscode.window.onDidChangeTextEditorVisibleRanges(event => { preview.editorScrolled(event.textEditor) }),
  )
}
