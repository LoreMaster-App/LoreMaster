import * as vscode from 'vscode'
import { type CheckCache, type CheckEngine, checkDiagrams } from './check-diagrams.use-case'
import type { CheckedLanguage } from './diagram-fence.contract'
import { locateDiagramFences } from './locate-diagram-fences.algorithm'

/** The setting that turns the checks off. */
export const DIAGRAM_DIAGNOSTICS_SETTING = 'loreMaster.diagrams.diagnostics'

/** How long to wait after the last keystroke before D2 is checked again. */
const TYPING_PAUSE_MS = 500

/** The most answers kept; the oldest go first. */
const CACHE_LIMIT = 300

/** What the checks need from the rest of the extension. */
export interface DiagramDiagnosticsDeps {
  engine: CheckEngine
  output: vscode.OutputChannel
}

/**
 * Puts the errors of Mermaid and D2 fences in the Problems panel, on the line of the diagram's source the language names. D2 is
 * checked as the file is edited (it is quick and exact); Mermaid is checked when a file is opened or saved, because the engine
 * starts a browser for it, and is left out with a note in the output when the machine has none.
 */
export function registerDiagramDiagnostics (deps: DiagramDiagnosticsDeps): vscode.Disposable {
  const collection = vscode.languages.createDiagnosticCollection('LoreMaster diagrams')
  const cache: CheckCache = new Map()
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  let said = false
  const enabled = (): boolean => vscode.workspace.getConfiguration().get<boolean>(DIAGRAM_DIAGNOSTICS_SETTING, true)

  async function check (document: vscode.TextDocument, ask: readonly CheckedLanguage[]): Promise<void> {
    if (document.languageId !== 'markdown' || !enabled()) { return }
    const fences = locateDiagramFences(document.getText())
    if (fences.length === 0) {
      collection.delete(document.uri)

      return
    }
    try {
      if (cache.size > CACHE_LIMIT) { cache.clear() }
      const { findings, unavailable } = await checkDiagrams(deps.engine, fences, cache, ask)
      collection.set(document.uri, findings.map(finding => {
        const line = Math.min(finding.line, document.lineCount - 1)
        const range = new vscode.Range(line, finding.column, line, document.lineAt(line).text.length)
        const diagnostic = new vscode.Diagnostic(range, finding.message, vscode.DiagnosticSeverity.Error)
        diagnostic.source = finding.language === 'd2' ? 'LoreMaster (D2)' : 'LoreMaster (Mermaid)'

        return diagnostic
      }))
      if (!said && unavailable.length > 0) {
        said = true
        deps.output.appendLine(`Diagram check: ${unavailable[0]}. Mermaid diagrams are not checked on this machine.`)
      }
    } catch (error) {
      deps.output.appendLine(`Diagram check failed: ${error instanceof Error ? error.message : String(error)}`)
    }
  }

  const later = (document: vscode.TextDocument): void => {
    const key = document.uri.toString()
    clearTimeout(timers.get(key))
    timers.set(key, setTimeout(() => {
      timers.delete(key)
      void check(document, ['d2'])
    }, TYPING_PAUSE_MS))
  }

  for (const document of vscode.workspace.textDocuments) { void check(document, ['d2', 'mermaid']) }

  return vscode.Disposable.from(
    collection,
    vscode.workspace.onDidOpenTextDocument(document => { void check(document, ['d2', 'mermaid']) }),
    vscode.workspace.onDidSaveTextDocument(document => { void check(document, ['d2', 'mermaid']) }),
    vscode.workspace.onDidChangeTextDocument(event => { if (event.contentChanges.length > 0) { later(event.document) } }),
    vscode.workspace.onDidCloseTextDocument(document => { collection.delete(document.uri) }),
    vscode.workspace.onDidChangeConfiguration(event => {
      if (!event.affectsConfiguration(DIAGRAM_DIAGNOSTICS_SETTING)) { return }
      if (enabled()) {
        for (const document of vscode.workspace.textDocuments) { void check(document, ['d2', 'mermaid']) }
      } else {
        collection.clear()
      }
    }),
    { dispose: () => { for (const timer of timers.values()) { clearTimeout(timer) } } },
  )
}
