import * as vscode from 'vscode'

const MARKDOWN_GLOB = '**/*.{md,markdown}'
const EXCLUDED = '**/{node_modules,.git}/**'

/** What an export command was invoked on, turned into the Markdown files to export. */
export async function markdownTargets (clicked: vscode.Uri | undefined, selected: readonly vscode.Uri[] | undefined): Promise<vscode.Uri[]> {
  const given = selected && selected.length > 0 ? [...selected] : (clicked ? [clicked] : activeMarkdown())
  const files: vscode.Uri[] = []
  for (const uri of given) {
    const stat = await statOf(uri)
    if (stat?.type === vscode.FileType.Directory) {
      files.push(...await vscode.workspace.findFiles(new vscode.RelativePattern(uri, MARKDOWN_GLOB), EXCLUDED))
    } else if (stat) {
      files.push(uri)
    }
  }

  return files.sort((left, right) => left.fsPath.localeCompare(right.fsPath))
}

async function statOf (uri: vscode.Uri): Promise<vscode.FileStat | null> {
  try {
    return await vscode.workspace.fs.stat(uri)
  } catch {
    return null
  }
}

function activeMarkdown (): vscode.Uri[] {
  const document = vscode.window.activeTextEditor?.document

  return document?.languageId === 'markdown' ? [document.uri] : []
}

/** The editor's text for a file when it holds changes that are not saved, so an export matches what is on screen. */
export function unsavedText (uri: vscode.Uri): string | undefined {
  const open = vscode.workspace.textDocuments.find(document => document.uri.fsPath === uri.fsPath)

  return open?.isDirty ? open.getText() : undefined
}
