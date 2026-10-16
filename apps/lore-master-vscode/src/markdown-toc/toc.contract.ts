/** How a table of contents is written. */
export interface TocOptions {
  /** The shallowest and deepest heading levels it lists. */
  minLevel: number
  maxLevel: number
  /** Numbered (`1.`) rather than bulleted (`-`). */
  ordered:  boolean
}

/** The marks that fence the part of a document a table of contents lives in. */
export const TOC_START = '<!-- toc -->'
export const TOC_END = '<!-- /toc -->'
