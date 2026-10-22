import * as path from 'node:path'
import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { ExportFormat } from '../engine-protocol'
import type { ExportOutcome, PrintOptions } from './export-markdown.contract'
import { exportDestination } from './export-destination.policy'
import { exportDocument, exportDocumentAsText, listExportFormats } from './export-document.use-case'
import { markdownTargets, unsavedText } from './export-targets.client'
import { formatPicks, textFormats } from './format-picks.mapper'

/** The command ids contributed in package.json. */
export const EXPORT_MARKDOWN_COMMAND = 'loreMaster.exportMarkdown'
export const COPY_MARKDOWN_AS_COMMAND = 'loreMaster.copyMarkdownAs'

/** What the export commands are given at registration. */
export interface ExportCommandDeps {
  engine:         EngineClient
  output:         vscode.OutputChannel
  /** Where the extension is installed, which is where the Mermaid library it ships is found. */
  extensionPath?: string
}

const OPEN = 'Open'
const SHOW = 'Show in folder'

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function setting<T> (key: string, fallback: T): T {
  return vscode.workspace.getConfiguration('loreMaster.export').get<T>(key, fallback)
}

/** The company templates the settings name, as absolute paths: a relative one is looked for in the workspace folder. */
function templatesIn (workspaceRoot: string | undefined): Record<string, string> {
  const templates: Record<string, string> = {}
  for (const [format, key] of [['docx', 'docxTemplate'], ['odt', 'odtTemplate']] as const) {
    const configured = setting<string>(key, '').trim()
    if (configured !== '') {
      templates[format] = path.isAbsolute(configured) ? configured : path.resolve(workspaceRoot ?? '.', configured)
    }
  }

  return templates
}

/** The settings that shape a PDF or a PNG; blank or unset ones are left to the engine's defaults. */
function printOptions (extensionPath: string | undefined, workspaceRoot: string | undefined): PrintOptions {
  return {
    templates:   templatesIn(workspaceRoot),
    mermaidPath: extensionPath ? path.join(extensionPath, 'dist', 'assets', 'webview', 'mermaid.min.js') : undefined,
    browserPath: setting<string>('browserPath', '').trim() || undefined,
    pageSize:    setting<string>('pageSize', '') || undefined,
    imageWidth:  setting<number>('pngWidth', 0) || undefined,
  }
}

/**
 * Exports the clicked or active Markdown file, the selected files, or every Markdown file in a
 * clicked folder, to the formats the person ticks.
 */
export async function exportMarkdownCommand (deps: ExportCommandDeps, clicked?: vscode.Uri, selected?: vscode.Uri[]): Promise<void> {
  const targets = await markdownTargets(clicked, selected)
  if (targets.length === 0) {
    await vscode.window.showInformationMessage('LoreMaster: open a Markdown file, or choose one or a folder in the explorer, to export.')

    return
  }

  let formats: ExportFormat[]
  try {
    formats = await listExportFormats(deps.engine)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${messageOf(error)}`)

    return
  }
  const picked = await vscode.window.showQuickPick(formatPicks(formats, setting<string[]>('formats', ['html'])), {
    canPickMany: true,
    title:       targets.length === 1 ? `LoreMaster: export ${vscode.workspace.asRelativePath(targets[0])}` : `LoreMaster: export ${targets.length} Markdown files`,
    placeHolder: 'Choose the formats to export to',
  })
  if (!picked || picked.length === 0) {
    return
  }

  const outcome = await vscode.window.withProgress({ location: vscode.ProgressLocation.Notification, title: 'LoreMaster: exporting' }, () => exportAll(deps, targets, picked.map(pick => pick.id)))
  await report(deps, outcome)
}

async function exportAll (deps: ExportCommandDeps, targets: readonly vscode.Uri[], formatIds: string[]): Promise<ExportOutcome> {
  const folder = setting<string>('folder', '')
  const total: ExportOutcome = { written: [], failed: [] }
  for (const target of targets) {
    const root = vscode.workspace.getWorkspaceFolder(target)?.uri.fsPath
    const outcome = await exportDocument(deps.engine, {
      path:    target.fsPath,
      content: unsavedText(target),
      formats: formatIds,
      outDir:  exportDestination(target.fsPath, root, folder),
      print:   printOptions(deps.extensionPath, root),
    })
    total.written.push(...outcome.written)
    total.failed.push(...outcome.failed.map(failure => ({ ...failure, message: `${vscode.workspace.asRelativePath(target)} (${failure.format}): ${failure.message}` })))
  }

  return total
}

async function report (deps: ExportCommandDeps, outcome: ExportOutcome): Promise<void> {
  for (const line of outcome.written) {
    deps.output.appendLine(`exported ${line.path}`)
  }
  for (const failure of outcome.failed) {
    deps.output.appendLine(`export failed: ${failure.message}`)
  }
  if (outcome.failed.length > 0) {
    deps.output.show(true)
    await vscode.window.showErrorMessage(`LoreMaster: ${outcome.failed.length} export(s) failed. ${outcome.failed[0].message}`)
  }
  if (outcome.written.length === 0) {
    return
  }

  const first = vscode.Uri.file(outcome.written[0].path)
  const summary = outcome.written.length === 1 ? `exported ${outcome.written[0].name}` : `exported ${outcome.written.length} files`
  const answer = await vscode.window.showInformationMessage(`LoreMaster: ${summary}.`, OPEN, SHOW)
  if (answer === OPEN) {
    await vscode.env.openExternal(first)
  } else if (answer === SHOW) {
    await vscode.commands.executeCommand('revealFileInOS', first)
  }
}

/**
 * Copies the active Markdown document, or its selection, to the clipboard in a text format, ready to
 * paste into Jira, a wiki or a message.
 */
export async function copyMarkdownAsCommand (deps: ExportCommandDeps): Promise<void> {
  const editor = vscode.window.activeTextEditor
  if (editor?.document.languageId !== 'markdown') {
    await vscode.window.showInformationMessage('LoreMaster: open a Markdown file to copy it.')

    return
  }

  try {
    const formats = textFormats(await listExportFormats(deps.engine))
    const picked = await vscode.window.showQuickPick(formatPicks(formats, []), {
      title:       editor.selection.isEmpty ? 'LoreMaster: copy the document as' : 'LoreMaster: copy the selection as',
      placeHolder: 'Choose a format',
    })
    if (!picked) {
      return
    }
    const content = editor.selection.isEmpty ? editor.document.getText() : editor.document.getText(editor.selection)
    const text = await exportDocumentAsText(deps.engine, { path: editor.document.uri.fsPath, content, format: picked.id })
    await vscode.env.clipboard.writeText(text)
    vscode.window.setStatusBarMessage(`LoreMaster: copied as ${picked.label}`, 4000)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${messageOf(error)}`)
  }
}
