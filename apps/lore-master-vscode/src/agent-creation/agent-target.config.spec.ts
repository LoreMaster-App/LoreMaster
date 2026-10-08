import { AGENT_TARGETS, BLOCK_BEGIN, BLOCK_END, FILE_MARKER, targetById } from './agent-target.config'

const instructions = [
  'You write documentation.',
  '',
  '## How LoreMaster organises pages',
  '',
  '1. **explicit-parent** — a `parent:` line.',
  '',
  '## How to work',
  '',
  '### A nested heading',
  '',
].join('\n')

describe('agent targets', () => {
  it('offers the three tools, each with its own file', () => {
    expect(AGENT_TARGETS.map(target => [target.id, target.path, target.kind])).toEqual([
      ['claude-code', '.claude/agents/loremaster-writer.md', 'file'],
      ['vscode-agent', '.github/agents/loremaster-writer.agent.md', 'file'],
      ['agents-md', 'AGENTS.md', 'block'],
    ])
  })

  it('writes a Claude Code subagent with its frontmatter, the MCP tools it may use, and the instructions', () => {
    const content = targetById('claude-code').render(instructions)

    expect(content.startsWith('---\nname: loremaster-writer\ndescription: ')).toBe(true)
    const frontmatter = content.split('---', 2)[1]
    expect(frontmatter).toContain('Use proactively whenever documentation is created, restructured or reviewed.')
    expect(frontmatter).toContain('tools: Read, Write, Edit, Glob, Grep, mcp__loremaster__loremaster_nesting_rules, mcp__loremaster__preview_tree, mcp__loremaster__place_document, mcp__loremaster__validate_document')
    expect(content).toContain(`${FILE_MARKER}\n\n# LoreMaster documentation writer\n\nYou write documentation.`)
    expect(content.endsWith('### A nested heading\n')).toBe(true)
  })

  it('writes a VS Code custom agent without a tools list, so it uses whatever tools the editor has', () => {
    const content = targetById('vscode-agent').render(instructions)

    expect(content.startsWith('---\nname: LoreMaster writer\ndescription: ')).toBe(true)
    expect(content.split('---', 2)[1]).not.toContain('tools:')
    expect(content).toContain(FILE_MARKER)
    expect(content).toContain('# LoreMaster documentation writer')
  })

  it('writes an AGENTS.md block with its own heading and the instructions one level down', () => {
    const block = targetById('agents-md').render(instructions)

    expect(block.startsWith(`${BLOCK_BEGIN}\n## Documentation (LoreMaster)\n\nYou write documentation.`)).toBe(true)
    expect(block).toContain('\n### How LoreMaster organises pages\n')
    expect(block).toContain('\n#### A nested heading\n')
    expect(block).not.toMatch(/^## How/m)
    expect(block.endsWith(`${BLOCK_END}\n`)).toBe(true)
  })

  it('never lets the instructions start a new top-level heading inside a block', () => {
    expect(targetById('agents-md').render('# Top\n\ntext\n')).toContain('\n## Top\n')
  })
})
