import type { ExportFormat } from '../engine-protocol'

/** One row of the format picker. */
export interface FormatPick {
  id:          string
  label:       string
  description: string
  picked:      boolean
}

/** The picker rows for a list of formats, with the preferred ones ticked. */
export function formatPicks (formats: readonly ExportFormat[], preferred: readonly string[]): FormatPick[] {
  return formats.map(format => ({
    id:          format.id,
    label:       format.label,
    description: format.extension,
    picked:      preferred.includes(format.id),
  }))
}

/** The formats whose result is text, which is what can be copied. */
export function textFormats (formats: readonly ExportFormat[]): ExportFormat[] {
  return formats.filter(format => !format.binary)
}
