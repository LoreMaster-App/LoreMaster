import { AGENT_INSTRUCTIONS_METHOD } from '../engine-protocol'
import { AGENT_TARGETS, FILE_MARKER, targetById } from './agent-target.config'
import { type AgentEngine, createAgent, type FileStore } from './create-agent.use-case'

function memoryFiles (initial: Record<string, string> = {}): FileStore & { files: Map<string, string> } {
  const files = new Map(Object.entries(initial))

  return {
    files,
    read:  path => Promise.resolve(files.get(path)),
    write: (path, content) => {
      files.set(path, content)

      return Promise.resolve()
    },
  }
}

function engine (instructions: string, hasConfig = true, requests: { method: string; params: unknown }[] = []): AgentEngine {
  return {
    request (method: string, params?: unknown) {
      requests.push({ method, params })

      return Promise.resolve({ instructions, hasConfig } as never)
    },
  }
}

const deps = (files: FileStore, answers: boolean[] = [], requests: { method: string; params: unknown }[] = []) => ({
  engine:           engine('RULES\n', true, requests),
  files,
  workspaceRoot:    '/w',
  confirmOverwrite: () => Promise.resolve(answers.shift() ?? false),
})

describe('createAgent', () => {
  it('asks the engine for the instructions of the workspace and writes each target', async () => {
    const requests: { method: string; params: unknown }[] = []
    const files = memoryFiles()

    const result = await createAgent(deps(files, [], requests), AGENT_TARGETS)

    expect(requests).toEqual([{ method: AGENT_INSTRUCTIONS_METHOD, params: { workspaceRoot: '/w' } }])
    expect(result.outcomes.map(entry => entry.outcome)).toEqual(['created', 'created', 'created'])
    expect(result.hasConfig).toBe(true)
    expect(files.files.get('.claude/agents/loremaster-writer.md')).toContain('RULES')
    expect(files.files.get('.github/agents/loremaster-writer.agent.md')).toContain('RULES')
    expect(files.files.get('AGENTS.md')).toContain('RULES')
  })

  it('writes only the chosen targets', async () => {
    const files = memoryFiles()

    await createAgent(deps(files), [targetById('vscode-agent')])

    expect(files.files.keys().toArray()).toEqual(['.github/agents/loremaster-writer.agent.md'])
  })

  it('updates what it wrote before, reports what is already current, and appends to a shared file', async () => {
    const files = memoryFiles()
    await createAgent(deps(files), AGENT_TARGETS)
    files.files.set('AGENTS.md', `# Our agents\n\n${files.files.get('AGENTS.md')}`)

    const same = await createAgent(deps(files), AGENT_TARGETS)
    expect(same.outcomes.map(entry => entry.outcome)).toEqual(['unchanged', 'unchanged', 'unchanged'])

    const changed = await createAgent({ ...deps(files), engine: engine('NEW RULES\n') }, AGENT_TARGETS)
    expect(changed.outcomes.map(entry => entry.outcome)).toEqual(['updated', 'updated', 'updated'])
    expect(files.files.get('AGENTS.md')?.startsWith('# Our agents\n\n')).toBe(true)
    expect(files.files.get('AGENTS.md')).toContain('NEW RULES')
    expect(files.files.get('AGENTS.md')).not.toContain('\nRULES\n')
  })

  it('appends its block to an AGENTS.md the user already has', async () => {
    const files = memoryFiles({ 'AGENTS.md': '# Agents\n\nBe kind.\n' })

    const result = await createAgent(deps(files), [targetById('agents-md')])

    expect(result.outcomes[0].outcome).toBe('appended')
    expect(files.files.get('AGENTS.md')?.startsWith('# Agents\n\nBe kind.\n\n')).toBe(true)
  })

  it('asks before replacing a file it did not write, and leaves it when the answer is no', async () => {
    const files = memoryFiles({ '.claude/agents/loremaster-writer.md': '# My own agent\n' })
    const asked: string[] = []

    const declined = await createAgent({
      ...deps(files),
      confirmOverwrite: path => {
        asked.push(path)

        return Promise.resolve(false)
      },
    }, [targetById('claude-code')])

    expect(asked).toEqual(['.claude/agents/loremaster-writer.md'])
    expect(declined.outcomes[0].outcome).toBe('skipped')
    expect(files.files.get('.claude/agents/loremaster-writer.md')).toBe('# My own agent\n')

    const accepted = await createAgent(deps(files, [true]), [targetById('claude-code')])

    expect(accepted.outcomes[0].outcome).toBe('updated')
    expect(files.files.get('.claude/agents/loremaster-writer.md')).toContain(FILE_MARKER)
  })

  it('reports that the instructions are general when the workspace has no config', async () => {
    const result = await createAgent({ ...deps(memoryFiles()), engine: engine('X\n', false) }, [targetById('claude-code')])

    expect(result.hasConfig).toBe(false)
  })

  it('writes nothing when the engine cannot give instructions', async () => {
    const files = memoryFiles()
    const failing: AgentEngine = { request: () => Promise.reject(new Error('settings are invalid')) }

    await expect(createAgent({ ...deps(files), engine: failing }, AGENT_TARGETS)).rejects.toThrow('settings are invalid')
    expect(files.files.size).toBe(0)
  })
})
