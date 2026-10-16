/** The one thing an export needs from the engine client: to make a request. */
export interface ExportEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** One document to export, to one or more formats. */
export interface ExportRequest {
  /** The Markdown file, absolute. */
  path:     string
  /** The editor's unsaved text, when it has some. */
  content?: string
  /** Format ids from the engine's list. */
  formats:  string[]
  /** Absolute folder to write into; the document's own folder when absent. */
  outDir?:  string
}

export interface ExportedFile {
  format: string
  name:   string
  path:   string
}

export interface ExportFailure {
  format:  string
  message: string
}

/** What an export produced: a failure in one format does not stop the others. */
export interface ExportOutcome {
  written: ExportedFile[]
  failed:  ExportFailure[]
}
