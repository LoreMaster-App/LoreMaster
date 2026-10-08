/** The places an agent can be written to: each tool reads its own file. */
export type AgentTargetId = 'claude-code' | 'vscode-agent' | 'agents-md'

/** A file the command writes, and how the engine's instructions are framed for it. */
export interface AgentTarget {
  id:          AgentTargetId
  label:       string
  description: string
  /** Workspace-relative, '/'-separated. */
  path:        string
  /** A whole file owned by the command, or a delimited block inside a file the user owns. */
  kind:        'file' | 'block'
  /** The file's content, or the block's, for these instructions. */
  render (instructions: string): string
}

/** Marks a whole file as written by the command, so running it again may update it in place. */
export const FILE_MARKER = '<!-- lore-master:agent managed by "LoreMaster: Create documentation agent"; run it again to refresh it -->'

/** Delimit the block the command owns inside a file such as AGENTS.md. */
export const BLOCK_BEGIN = '<!-- lore-master:agent begin (managed by "LoreMaster: Create documentation agent"; edit outside this block) -->'
export const BLOCK_END = '<!-- lore-master:agent end -->'

const DESCRIPTION = 'Writes and edits this project\'s documentation as Markdown that LoreMaster syncs, placing, naming and titling every page the way the sync expects. Use proactively whenever documentation is created, restructured or reviewed.'

/** The LoreMaster MCP tools, named the way Claude Code addresses a server called "loremaster". */
const MCP_TOOLS = ['loremaster_nesting_rules', 'preview_tree', 'place_document', 'validate_document'].map(tool => `mcp__loremaster__${tool}`)

const heading = '# LoreMaster documentation writer'

/** One heading level deeper, so the instructions sit under a heading of their own. Only lines
 *  that start with # are touched; the instructions contain no fenced code. */
function demoteHeadings (markdown: string): string {
  return markdown.replaceAll(/^(#{1,5}) /gm, '$1# ')
}

export const AGENT_TARGETS: readonly AgentTarget[] = [
  {
    id:          'claude-code',
    label:       'Claude Code subagent',
    description: '.claude/agents/loremaster-writer.md',
    path:        '.claude/agents/loremaster-writer.md',
    kind:        'file',
    render:      instructions => [
      '---',
      'name: loremaster-writer',
      `description: ${DESCRIPTION}`,
      `tools: ${['Read', 'Write', 'Edit', 'Glob', 'Grep', ...MCP_TOOLS].join(', ')}`,
      '---',
      '',
      FILE_MARKER,
      '',
      heading,
      '',
      instructions.trimEnd(),
      '',
    ].join('\n'),
  },
  {
    id:          'vscode-agent',
    label:       'VS Code custom agent',
    description: '.github/agents/loremaster-writer.agent.md',
    path:        '.github/agents/loremaster-writer.agent.md',
    kind:        'file',
    render:      instructions => [
      '---',
      'name: LoreMaster writer',
      `description: ${DESCRIPTION}`,
      '---',
      '',
      FILE_MARKER,
      '',
      heading,
      '',
      instructions.trimEnd(),
      '',
    ].join('\n'),
  },
  {
    id:          'agents-md',
    label:       'AGENTS.md block',
    description: 'for any agent that reads AGENTS.md',
    path:        'AGENTS.md',
    kind:        'block',
    render:      instructions => [
      BLOCK_BEGIN,
      '## Documentation (LoreMaster)',
      '',
      demoteHeadings(instructions.trimEnd()),
      BLOCK_END,
      '',
    ].join('\n'),
  },
]

export function targetById (id: AgentTargetId): AgentTarget {
  const found = AGENT_TARGETS.find(target => target.id === id)
  if (!found) {
    throw new Error(`unknown agent target ${id}`)
  }

  return found
}
