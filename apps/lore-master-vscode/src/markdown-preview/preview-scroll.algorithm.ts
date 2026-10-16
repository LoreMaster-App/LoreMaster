const clamp = (value: number): number => Math.min(Math.max(value, 0), 1)

/** How far down a document a line is, from 0 for the first line to 1 for the last. */
export function fractionOfLine (line: number, lineCount: number): number {
  return lineCount <= 1 ? 0 : clamp(line / (lineCount - 1))
}

/** The line that is a fraction of the way down a document. */
export function lineOfFraction (fraction: number, lineCount: number): number {
  return lineCount <= 1 ? 0 : Math.round(clamp(fraction) * (lineCount - 1))
}
