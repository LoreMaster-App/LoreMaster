import * as vscode from 'vscode'
import { applyEditResults, documentLines, selectionsOf, type EditResult } from '../text-edits'
import { applyFormatting, type FormattingAction } from './apply-formatting.algorithm'

/** The command ids contributed in package.json, by what they do. */
export const FORMATTING_COMMANDS: Record<FormattingAction, string> = {
  bold:            'loreMaster.markdown.toggleBold',
  italic:          'loreMaster.markdown.toggleItalic',
  strikethrough:   'loreMaster.markdown.toggleStrikethrough',
  inlineCode:      'loreMaster.markdown.toggleInlineCode',
  codeBlock:       'loreMaster.markdown.toggleCodeBlock',
  quote:           'loreMaster.markdown.toggleQuote',
  task:            'loreMaster.markdown.toggleTask',
  increaseHeading: 'loreMaster.markdown.increaseHeading',
  decreaseHeading: 'loreMaster.markdown.decreaseHeading',
}

/** Does one formatting action at every cursor of the active Markdown editor. */
export async function formatMarkdown (action: FormattingAction): Promise<void> {
  const editor = vscode.window.activeTextEditor
  if (editor?.document.languageId !== 'markdown') {
    return
  }

  const lines = documentLines(editor.document)
  const results = selectionsOf(editor)
    .map(selection => applyFormatting(action, lines, selection))
    .filter((result): result is EditResult => result !== undefined)
  await applyEditResults(editor, results)
}
