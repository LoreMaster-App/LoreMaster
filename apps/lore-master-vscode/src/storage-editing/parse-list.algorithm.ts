/** A comma- or line-separated list, trimmed, without empty entries. */
export function parseList (value: string): string[] {
  return value.split(/[,\n]/).map(entry => entry.trim()).filter(entry => entry !== '')
}
