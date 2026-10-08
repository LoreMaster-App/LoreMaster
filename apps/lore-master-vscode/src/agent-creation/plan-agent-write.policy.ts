import { type AgentTarget, FILE_MARKER } from './agent-target.config'

/** What writing a target would do to the file that is there now. */
export type WriteAction = 'create' | 'update' | 'unchanged' | 'append' | 'conflict'

export interface WritePlan {
  action:   WriteAction
  /** The whole file to write; absent for 'unchanged' and 'conflict'. */
  content?: string
}

/** The block the command owns in a shared file, from its begin line to its end line. */
const BLOCK_PATTERN = /<!-- lore-master:agent begin[^>]*-->[\s\S]*?<!-- lore-master:agent end -->\n?/

/**
 * Decides how to write a target without losing anything the user wrote:
 *
 * - a whole file the command owns is created, and updated in place only when it still carries
 *   the command's marker; a file without it is a conflict, left for the user to decide;
 * - a block in a shared file (AGENTS.md) replaces the previous block, or is appended after the
 *   user's own content, which is never touched.
 */
export function planAgentWrite (target: AgentTarget, existing: string | undefined, rendered: string): WritePlan {
  if (existing === undefined) {
    return { action: 'create', content: rendered }
  }

  if (target.kind === 'file') {
    if (!existing.includes(FILE_MARKER)) {
      return { action: 'conflict' }
    }

    return existing === rendered ? { action: 'unchanged' } : { action: 'update', content: rendered }
  }

  if (BLOCK_PATTERN.test(existing)) {
    const replaced = existing.replace(BLOCK_PATTERN, () => rendered)

    return replaced === existing ? { action: 'unchanged' } : { action: 'update', content: replaced }
  }

  return { action: 'append', content: existing + separatorFor(existing) + rendered }
}

/** What goes between the user's content and an appended block: a blank line, no more. */
function separatorFor (existing: string): string {
  if (existing === '' || existing.endsWith('\n\n')) {
    return ''
  }

  return existing.endsWith('\n') ? '\n' : '\n\n'
}
