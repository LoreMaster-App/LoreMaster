import { EXPORT_FORMATS_METHOD, EXPORT_RUN_METHOD, type ExportFormat, type ExportFormatsResult, type ExportRunParams, type ExportRunResult } from '../engine-protocol'
import type { ExportEngine, ExportOutcome, ExportRequest } from './export-markdown.contract'

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

/** The formats the engine can export to. */
export async function listExportFormats (engine: ExportEngine): Promise<ExportFormat[]> {
  const result = await engine.request<ExportFormatsResult>(EXPORT_FORMATS_METHOD)

  return result.formats
}

/** Exports one document to each of the requested formats, one file per format. */
export async function exportDocument (engine: ExportEngine, request: ExportRequest): Promise<ExportOutcome> {
  const outcome: ExportOutcome = { written: [], failed: [] }
  for (const format of request.formats) {
    const params: ExportRunParams = { path: request.path, format, content: request.content, outDir: request.outDir, ...request.print }
    try {
      const result = await engine.request<ExportRunResult>(EXPORT_RUN_METHOD, params)
      outcome.written.push({ format, name: result.name, path: result.path ?? '' })
    } catch (error) {
      outcome.failed.push({ format, message: messageOf(error) })
    }
  }

  return outcome
}

/** The text of a document (or of a piece of it) in a text format, to put on the clipboard. */
export async function exportDocumentAsText (engine: ExportEngine, request: { path: string, content?: string, format: string, preview?: boolean }): Promise<string> {
  const result = await engine.request<ExportRunResult>(EXPORT_RUN_METHOD, { path: request.path, content: request.content, format: request.format, inline: true, preview: request.preview } satisfies ExportRunParams)

  return result.text ?? ''
}
