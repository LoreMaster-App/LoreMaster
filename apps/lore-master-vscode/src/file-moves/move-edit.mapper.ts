import type { FilesPlanMoveResult } from '../engine-protocol'

/** One step of the edit that keeps nesting intact, in the order it must be applied. */
export type EditOperation =
  | { kind: 'replace-line'; path: string; line: number; text: string } |
  { kind: 'rename'; from: string; to: string }

/** The plan as steps: text edits first, at the files' current paths, then the renames. */
export function editOperations (plan: FilesPlanMoveResult): EditOperation[] {
  return [
    ...plan.edits.map((edit): EditOperation => ({ kind: 'replace-line', path: edit.path, line: edit.line, text: edit.text })),
    ...plan.renames.map((rename): EditOperation => ({ kind: 'rename', from: rename.from, to: rename.to })),
  ]
}

function count (n: number, noun: string): string {
  return `${n} ${noun}${n === 1 ? '' : 's'}`
}

/** What the plan does, in a sentence for the question: "rename 2 files and update 1 parent line". */
export function summarise (plan: FilesPlanMoveResult): string {
  const parts = [
    plan.renames.length > 0 ? `rename ${count(plan.renames.length, 'file')} that nest${plan.renames.length === 1 ? 's' : ''} under it` : '',
    plan.edits.length > 0 ? `update ${count(plan.edits.length, '`parent:` line')}` : '',
  ].filter(part => part !== '')

  return parts.join(' and ')
}
