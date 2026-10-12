import { join } from 'node:path'
import * as vscode from 'vscode'
import type { FileMove, FilesPlanMoveResult } from '../engine-protocol'
import { editOperations, summarise } from './move-edit.mapper'
import { afterMoves, mayHoldMarkdown, workspaceRelative } from './move-paths.algorithm'
import { type FileMovesEngine, planMove } from './plan-move.use-case'

/** The setting that says whether moving a file updates what nests under it. */
export const UPDATE_ON_MOVE_SETTING = 'pages.updateOnMove'

type Mode = 'prompt' | 'always' | 'never'

interface PlannedMove {
  root:  string
  moves: FileMove[]
  plan:  FilesPlanMoveResult
}

function modeFrom (value: unknown): Mode {
  return value === 'always' || value === 'never' ? value : 'prompt'
}

function keyOf (files: readonly { oldUri: vscode.Uri }[]): string {
  return files.map(file => file.oldUri.toString()).join('|')
}

async function isFolder (uri: vscode.Uri): Promise<boolean> {
  try {
    const { type } = await vscode.workspace.fs.stat(uri)

    return (type & vscode.FileType.Directory) !== 0
  } catch {
    return false
  }
}

/**
 * Keeps nesting intact when Markdown files are moved or renamed in the editor: the files that
 * nest under a moved one by name follow it, and `parent:` lines that name it are rewritten.
 * The plan is made before the move, while the old paths still exist, and applied after it, so
 * the question to the user cannot hold the move up. Links are VS Code's own job
 * (`markdown.updateLinksOnFileMove.enabled`).
 */
export function registerFileMoves (deps: { engine: FileMovesEngine }): vscode.Disposable {
  const planned = new Map<string, PlannedMove[]>()
  /** Paths this handler is renaming itself; their own rename events are not planned again. */
  const ours = new Set<string>()
  const mode = (): Mode => modeFrom(vscode.workspace.getConfiguration('loreMaster').get(UPDATE_ON_MOVE_SETTING))

  const plan = async (files: readonly { oldUri: vscode.Uri; newUri: vscode.Uri }[]): Promise<undefined> => {
    if (mode() === 'never') {
      return
    }
    const byRoot: Record<string, FileMove[]> = {}
    for (const file of files) {
      if (file.oldUri.scheme !== 'file' || ours.delete(file.oldUri.fsPath.toLowerCase())) {
        continue
      }
      const folder = vscode.workspace.getWorkspaceFolder(file.oldUri)
      if (!folder || vscode.workspace.getWorkspaceFolder(file.newUri)?.uri.toString() !== folder.uri.toString()) {
        continue
      }
      const from = workspaceRelative(folder.uri.fsPath, file.oldUri.fsPath)
      const to = workspaceRelative(folder.uri.fsPath, file.newUri.fsPath)
      if (from === undefined || to === undefined || !mayHoldMarkdown(from, await isFolder(file.oldUri))) {
        continue
      }
      byRoot[folder.uri.fsPath] = [...byRoot[folder.uri.fsPath] ?? [], { from, to }]
    }

    const results: PlannedMove[] = []
    for (const [root, moves] of Object.entries(byRoot)) {
      const result = await planMove(deps.engine, root, moves)
      if (result && (result.renames.length > 0 || result.edits.length > 0 || result.warnings.length > 0)) {
        results.push({ root, moves, plan: result })
      }
    }
    if (results.length > 0) {
      planned.set(keyOf(files), results)
    }
  }

  const apply = async (found: PlannedMove): Promise<void> => {
    const { root, moves, plan: result } = found
    const edit = new vscode.WorkspaceEdit()
    for (const operation of editOperations(result)) {
      if (operation.kind === 'rename') {
        const from = vscode.Uri.file(join(root, operation.from))
        ours.add(from.fsPath.toLowerCase())
        edit.renameFile(from, vscode.Uri.file(join(root, operation.to)))

        continue
      }
      const uri = vscode.Uri.file(join(root, afterMoves(operation.path, moves)))
      const document = await vscode.workspace.openTextDocument(uri)
      if (operation.line < document.lineCount) {
        edit.replace(uri, document.lineAt(operation.line).range, operation.text)
      }
    }
    try {
      await vscode.workspace.applyEdit(edit)
    } finally {
      ours.clear()
    }
  }

  const ask = async (found: PlannedMove): Promise<boolean> => {
    if (mode() === 'always') {
      return true
    }
    const question = `LoreMaster: ${summarise(found.plan)}?`
    const answer = await vscode.window.showInformationMessage(question, 'Update', 'Always', 'Not now')
    if (answer === 'Always') {
      await vscode.workspace.getConfiguration('loreMaster').update(UPDATE_ON_MOVE_SETTING, 'always', vscode.ConfigurationTarget.Global)
    }

    return answer === 'Update' || answer === 'Always'
  }

  return vscode.Disposable.from(
    vscode.workspace.onWillRenameFiles(event => {
      event.waitUntil(plan(event.files))
    }),
    vscode.workspace.onDidRenameFiles(async event => {
      const found = planned.get(keyOf(event.files))
      planned.delete(keyOf(event.files))
      const work = found ?? []
      for (const each of work) {
        for (const warning of each.plan.warnings) {
          void vscode.window.showWarningMessage(`LoreMaster: ${warning}`)
        }
        if ((each.plan.renames.length > 0 || each.plan.edits.length > 0) && await ask(each)) {
          await apply(each)
        }
      }
    }),
  )
}
