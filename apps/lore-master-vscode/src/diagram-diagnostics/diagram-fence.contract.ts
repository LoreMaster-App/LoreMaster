/** The languages whose fences are checked. */
export type CheckedLanguage = 'mermaid' | 'd2'

/** A diagram fence of a Markdown document. */
export interface DiagramFence {
  language:  CheckedLanguage
  /** The diagram's text, without the fence lines. */
  source:    string
  /** The 0-based line of the first line of the source: the line after the opening fence. */
  firstLine: number
}

/** One thing wrong with a diagram, placed in the document. */
export interface DiagramFinding {
  language: CheckedLanguage
  /** 0-based line in the document. */
  line:     number
  /** 0-based column the problem starts at; 0 when the language does not say. */
  column:   number
  message:  string
}
