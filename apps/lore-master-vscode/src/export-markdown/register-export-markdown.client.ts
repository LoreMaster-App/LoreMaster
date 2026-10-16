import * as vscode from 'vscode'
import { COPY_MARKDOWN_AS_COMMAND, copyMarkdownAsCommand, EXPORT_MARKDOWN_COMMAND, exportMarkdownCommand, type ExportCommandDeps } from './export-markdown.handler'

/** Registers the export and copy-as commands. */
export function registerExportMarkdown (deps: ExportCommandDeps): vscode.Disposable {
  return vscode.Disposable.from(
    vscode.commands.registerCommand(EXPORT_MARKDOWN_COMMAND, (clicked?: vscode.Uri, selected?: vscode.Uri[]) => exportMarkdownCommand(deps, clicked, selected)),
    vscode.commands.registerCommand(COPY_MARKDOWN_AS_COMMAND, () => copyMarkdownAsCommand(deps)),
  )
}
