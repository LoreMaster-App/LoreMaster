/** A place in a document, zero-based, in UTF-16 units as VS Code counts them. */
export interface TextPosition {
  line:      number
  character: number
}

export interface TextRange {
  start: TextPosition
  end:   TextPosition
}

/** Replace a range with text. An insertion is an empty range. */
export interface TextEdit {
  range: TextRange
  text:  string
}

export interface TextSelection {
  anchor: TextPosition
  active: TextPosition
}

/**
 * What an editing command wants done for one cursor: the edits, against the document as it is, and where
 * that cursor ends up once they are applied.
 */
export interface EditResult {
  edits:     TextEdit[]
  selection: TextSelection
}
