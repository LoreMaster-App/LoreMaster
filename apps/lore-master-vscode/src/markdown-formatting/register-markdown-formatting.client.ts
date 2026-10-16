import * as vscode from 'vscode'
import type { FormattingAction } from './apply-formatting.algorithm'
import { FORMATTING_COMMANDS, formatMarkdown } from './format-markdown.handler'

/** Registers the formatting commands. */
export function registerMarkdownFormatting (): vscode.Disposable {
  const actions = Object.keys(FORMATTING_COMMANDS) as FormattingAction[]

  return vscode.Disposable.from(...actions.map(action => vscode.commands.registerCommand(FORMATTING_COMMANDS[action], () => formatMarkdown(action))))
}
