/** A list item written as a task: its bullet or number (in a quote too), then `[`, the state, `]`. */
const TASK_LINE = /^((?:[ \t]*>)*[ \t]*(?:[-*+]|\d+[.)])[ \t]+\[)([ x])(\])/i

/** The line with its task flipped (`[ ]` to `[x]`, `[x]` or `[X]` to `[ ]`), or undefined when the line is not a task list item. */
export function toggleTaskInLine (line: string): string | undefined {
  const match = TASK_LINE.exec(line)
  if (match === null) { return undefined }

  return `${match[1]}${match[2] === ' ' ? 'x' : ' '}${match[3]}${line.slice(match[0].length)}`
}
