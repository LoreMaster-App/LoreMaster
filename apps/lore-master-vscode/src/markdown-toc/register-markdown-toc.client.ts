import * as vscode from 'vscode'
import {
  ADD_SECTION_NUMBERS_COMMAND, addSectionNumbersCommand, INSERT_TOC_COMMAND, insertTocCommand, REMOVE_SECTION_NUMBERS_COMMAND,
  removeSectionNumbersCommand, tocSaveEdits, UPDATE_TOC_COMMAND, updateTocCommand,
} from './toc-commands.handler'

/** Registers the table of contents and section number commands, and keeps a file's table of contents current as it is saved. */
export function registerMarkdownToc (): vscode.Disposable {
  return vscode.Disposable.from(
    vscode.commands.registerCommand(INSERT_TOC_COMMAND, insertTocCommand),
    vscode.commands.registerCommand(UPDATE_TOC_COMMAND, updateTocCommand),
    vscode.commands.registerCommand(ADD_SECTION_NUMBERS_COMMAND, addSectionNumbersCommand),
    vscode.commands.registerCommand(REMOVE_SECTION_NUMBERS_COMMAND, removeSectionNumbersCommand),
    vscode.workspace.onWillSaveTextDocument(event => {
      if (event.document.languageId === 'markdown') {
        event.waitUntil(Promise.resolve(tocSaveEdits(event.document)))
      }
    }),
  )
}
