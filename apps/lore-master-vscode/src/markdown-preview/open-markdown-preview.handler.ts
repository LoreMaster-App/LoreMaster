import * as vscode from 'vscode'
import { listExportFormats, textFormats } from '../export-markdown'
import type { EngineClient } from '../engine-process'
import type { MarkdownPreview } from './markdown-preview.client'
import { RENDERED } from './markdown-preview.client'

/** The command ids contributed in package.json. */
export const OPEN_PREVIEW_COMMAND = 'loreMaster.markdown.openPreview'
export const PREVIEW_AS_COMMAND = 'loreMaster.markdown.previewAs'

function activeMarkdown (): vscode.TextDocument | undefined {
  const document = vscode.window.activeTextEditor?.document

  return document?.languageId === 'markdown' ? document : undefined
}

/** Opens the preview of the active Markdown file beside it. */
export async function openPreview (preview: MarkdownPreview): Promise<void> {
  const document = activeMarkdown()
  if (!document) {
    await vscode.window.showInformationMessage('LoreMaster: open a Markdown file to preview it.')

    return
  }
  preview.showAs(RENDERED, 'Rendered')
  preview.show(document)
}

/** Previews the active file as the text of another format, or as the rendered page. */
export async function previewAs (preview: MarkdownPreview, engine: EngineClient): Promise<void> {
  const document = activeMarkdown()
  if (!document) {
    await vscode.window.showInformationMessage('LoreMaster: open a Markdown file to preview it.')

    return
  }
  try {
    const formats = textFormats(await listExportFormats(engine)).filter(format => format.id !== RENDERED)
    const picks = [{ id: RENDERED, label: 'Rendered page', description: 'the page as it reads' }, ...formats.map(format => ({ id: format.id, label: format.label, description: format.extension }))]
    const picked = await vscode.window.showQuickPick(picks, { title: 'Preview as', placeHolder: 'How should the preview show this file?' })
    if (!picked) {
      return
    }
    preview.showAs(picked.id, picked.label)
    preview.show(document)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}
