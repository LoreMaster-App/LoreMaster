import * as vscode from 'vscode'
import { applyEditResults, documentLines, selectionsOf, type EditResult } from '../text-edits'
import { continueList } from './continue-list.algorithm'
import { indentListItems } from './indent-list.algorithm'

/** The command ids contributed in package.json. */
export const LIST_ENTER_COMMAND = 'loreMaster.markdown.listEnter'
export const LIST_INDENT_COMMAND = 'loreMaster.markdown.listIndent'
export const LIST_OUTDENT_COMMAND = 'loreMaster.markdown.listOutdent'

/** The context key that says the cursor is on a list item, so the keys below only take over there. */
export const IN_LIST_CONTEXT = 'loreMaster.markdown.inList'

/** Runs a list edit for every cursor; when any cursor has nothing to do, the key does what it always does. */
async function editList (work: (lines: string[], selection: ReturnType<typeof selectionsOf>[number]) => EditResult | undefined, fallback: () => Thenable<unknown>): Promise<void> {
  const editor = vscode.window.activeTextEditor
  if (editor?.document.languageId !== 'markdown') {
    await fallback()

    return
  }
  const lines = documentLines(editor.document)
  const selections = selectionsOf(editor)
  const results = selections.map(selection => work(lines, selection))
  if (results.includes(undefined)) {
    await fallback()

    return
  }
  await applyEditResults(editor, results as EditResult[])
}

/** Enter on a list item: the next item, or the end of the list. */
export function listEnter (): Promise<void> {
  return editList(
    (lines, selection) => (selection.anchor.line === selection.active.line && selection.anchor.character === selection.active.character ? continueList(lines, selection.active) : undefined),
    () => vscode.commands.executeCommand('type', { text: '\n' }),
  )
}

/** Tab on a list item: nest it one level deeper. */
export function listIndent (): Promise<void> {
  return editList((lines, selection) => indentListItems(lines, selection, 'indent'), () => vscode.commands.executeCommand('tab'))
}

/** Shift+Tab on a list item: bring it one level out. */
export function listOutdent (): Promise<void> {
  return editList((lines, selection) => indentListItems(lines, selection, 'outdent'), () => vscode.commands.executeCommand('outdent'))
}
