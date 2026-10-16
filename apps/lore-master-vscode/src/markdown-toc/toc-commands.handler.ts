import * as vscode from 'vscode'
import { documentLines, type TextEdit } from '../text-edits'
import { addSectionNumbers, removeSectionNumbers } from './number-sections.algorithm'
import type { TocOptions } from './toc.contract'
import { hasToc, insertToc, updateToc } from './update-toc.algorithm'

/** The command ids contributed in package.json. */
export const INSERT_TOC_COMMAND = 'loreMaster.markdown.insertToc'
export const UPDATE_TOC_COMMAND = 'loreMaster.markdown.updateToc'
export const ADD_SECTION_NUMBERS_COMMAND = 'loreMaster.markdown.addSectionNumbers'
export const REMOVE_SECTION_NUMBERS_COMMAND = 'loreMaster.markdown.removeSectionNumbers'

const settings = (): vscode.WorkspaceConfiguration => vscode.workspace.getConfiguration('loreMaster.markdown.toc')

/** How a table of contents is written, from the settings; a minimum deeper than the maximum is brought back in line. */
export function tocOptions (): TocOptions {
  const minLevel = Math.min(Math.max(settings().get<number>('minLevel', 2), 1), 6)
  const maxLevel = Math.min(Math.max(settings().get<number>('maxLevel', 4), minLevel), 6)

  return { minLevel, maxLevel, ordered: settings().get<boolean>('ordered', false) }
}

const toRange = (edit: TextEdit): vscode.Range => new vscode.Range(edit.range.start.line, edit.range.start.character, edit.range.end.line, edit.range.end.character)

function activeMarkdown (): vscode.TextEditor | undefined {
  const editor = vscode.window.activeTextEditor

  return editor?.document.languageId === 'markdown' ? editor : undefined
}

async function apply (editor: vscode.TextEditor, edits: readonly TextEdit[]): Promise<void> {
  if (edits.length === 0) {
    return
  }
  await editor.edit(builder => {
    for (const edit of edits) {
      builder.replace(toRange(edit), edit.text)
    }
  })
}

/** Writes a table of contents at the cursor, or brings the one the file has up to date. */
export async function insertTocCommand (): Promise<void> {
  const editor = activeMarkdown()
  if (!editor) {
    return
  }
  const edit = insertToc(documentLines(editor.document), { line: editor.selection.active.line, character: editor.selection.active.character }, tocOptions())
  await apply(editor, edit ? [edit] : [])
}

/** Brings the table of contents of the active file up to date. */
export async function updateTocCommand (): Promise<void> {
  const editor = activeMarkdown()
  if (!editor) {
    return
  }
  const lines = documentLines(editor.document)
  if (!hasToc(lines)) {
    await vscode.window.showInformationMessage('LoreMaster: this file has no table of contents. Use "Insert Table of Contents" to add one.')

    return
  }
  const edit = updateToc(lines, tocOptions())
  await apply(editor, edit ? [edit] : [])
}

/** Numbers the headings, then keeps the table of contents (if there is one) in step with the new titles. */
export async function addSectionNumbersCommand (): Promise<void> {
  const editor = activeMarkdown()
  if (!editor) {
    return
  }
  await apply(editor, addSectionNumbers(documentLines(editor.document), tocOptions()))
  await updateTocQuietly(editor)
}

/** Takes the numbers off the headings, then keeps the table of contents in step. */
export async function removeSectionNumbersCommand (): Promise<void> {
  const editor = activeMarkdown()
  if (!editor) {
    return
  }
  await apply(editor, removeSectionNumbers(documentLines(editor.document), tocOptions()))
  await updateTocQuietly(editor)
}

async function updateTocQuietly (editor: vscode.TextEditor): Promise<void> {
  const edit = updateToc(documentLines(editor.document), tocOptions())
  await apply(editor, edit ? [edit] : [])
}

/** The edit that brings a document's table of contents up to date as it is saved, when that is switched on and it has one. */
export function tocSaveEdits (document: vscode.TextDocument): vscode.TextEdit[] {
  if (!settings().get<boolean>('updateOnSave', true)) {
    return []
  }
  const edit = updateToc(documentLines(document), tocOptions())

  return edit ? [vscode.TextEdit.replace(toRange(edit), edit.text)] : []
}
