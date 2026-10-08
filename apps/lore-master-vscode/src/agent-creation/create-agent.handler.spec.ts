import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { AGENT_INSTRUCTIONS_METHOD } from '../engine-protocol'
import { createAgentCommand } from './create-agent.handler'
import type { FileStore } from './create-agent.use-case'

interface Scenario {
  messages:   string[]
  buttons:    string[][]
  answers:    (string | undefined)[]
  quickPicks: ((items: { label: string; target: unknown }[]) => unknown)[]
  executed:   unknown[][]
  requests:   string[]
}

const window = vscode.window as unknown as Record<string, unknown>
const commands = vscode.commands as unknown as Record<string, unknown>
const original = { ...window }
const originalExecute = commands.executeCommand

function script (scenario: Scenario): void {
  window.showInformationMessage = (message: string, ...buttons: string[]) => {
    scenario.messages.push(`info: ${message}`)
    scenario.buttons.push(buttons)

    return Promise.resolve(scenario.answers.shift())
  }
  window.showErrorMessage = (message: string) => {
    scenario.messages.push(`error: ${message}`)

    return Promise.resolve(undefined)
  }
  window.showWarningMessage = (message: string) => {
    scenario.messages.push(`warn: ${message}`)

    return Promise.resolve(scenario.answers.shift())
  }
  window.showQuickPick = (items: { label: string; target: unknown }[]) => Promise.resolve(scenario.quickPicks.shift()?.(items))
  commands.executeCommand = (...arguments_: unknown[]) => {
    scenario.executed.push(arguments_)

    return Promise.resolve(undefined)
  }
}

function engine (scenario: Scenario, fail?: string): EngineClient {
  return {
    request (method: string) {
      scenario.requests.push(method)
      if (fail) {
        return Promise.reject(new Error(fail))
      }

      return Promise.resolve({ instructions: 'RULES\n', hasConfig: true } as never)
    },
  } as unknown as EngineClient
}

function memory (initial: Record<string, string> = {}): { files: Map<string, string>; factory: (folder: string) => FileStore } {
  const files = new Map(Object.entries(initial))

  return {
    files,
    factory: () => ({
      read:  path => Promise.resolve(files.get(path)),
      write: (path, content) => {
        files.set(path, content)

        return Promise.resolve()
      },
    }),
  }
}

function setFolders (folders: { uri: { fsPath: string }; name: string }[] | undefined): void {
  (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = folders
}

const pickAll = (items: { label: string; target: unknown }[]): unknown => items
const pick = (...labels: string[]) => (items: { label: string; target: unknown }[]): unknown => items.filter(item => labels.includes(item.label))

describe('createAgentCommand', () => {
  let scenario: Scenario
  beforeEach(() => {
    scenario = { messages: [], buttons: [], answers: [], quickPicks: [], executed: [], requests: [] }
    script(scenario)
    setFolders([{ uri: { fsPath: '/w' }, name: 'w' }])
  })
  afterEach(() => {
    Object.assign(window, original)
    commands.executeCommand = originalExecute
    setFolders(undefined)
  })

  it('offers the file targets preselected and the shared AGENTS.md block not', async () => {
    let offered: { label: string; picked?: boolean }[] = []
    scenario.quickPicks.push(items => {
      offered = items

      return undefined
    })

    await createAgentCommand({ engine: engine(scenario), files: memory().factory })

    expect(offered.map(item => [item.label, item.picked])).toEqual([['Claude Code subagent', true], ['VS Code custom agent', true], ['AGENTS.md block', false]])
  })

  it('writes the picked targets, says what it did, and points at the MCP server', async () => {
    const store = memory()
    scenario.quickPicks.push(pick('Claude Code subagent'))

    await createAgentCommand({ engine: engine(scenario), files: store.factory })

    expect(scenario.requests).toEqual([AGENT_INSTRUCTIONS_METHOD])
    expect(store.files.keys().toArray()).toEqual(['.claude/agents/loremaster-writer.md'])
    expect(scenario.messages).toEqual(['info: LoreMaster agent: created .claude/agents/loremaster-writer.md. For the place and validate tools, add the LoreMaster MCP server.'])
    expect(scenario.buttons).toEqual([['Open', 'Copy MCP Server Config']])
  })

  it('opens the written file when asked', async () => {
    scenario.quickPicks.push(pick('VS Code custom agent'))
    scenario.answers.push('Open')

    await createAgentCommand({ engine: engine(scenario), files: memory().factory })

    expect(scenario.executed).toHaveLength(1)
    expect(scenario.executed[0][0]).toBe('vscode.open')
    expect((scenario.executed[0][1] as { fsPath: string }).fsPath).toBe('/w/.github/agents/loremaster-writer.agent.md')
  })

  it('copies the MCP server config when asked', async () => {
    scenario.quickPicks.push(pick('AGENTS.md block'))
    scenario.answers.push('Copy MCP Server Config')

    await createAgentCommand({ engine: engine(scenario), files: memory().factory })

    expect(scenario.executed).toEqual([['loreMaster.copyMcpConfig']])
  })

  it('asks before replacing a file it did not write, and keeps the file when declined', async () => {
    const store = memory({ '.claude/agents/loremaster-writer.md': '# Mine\n' })
    scenario.quickPicks.push(pick('Claude Code subagent'))

    await createAgentCommand({ engine: engine(scenario), files: store.factory })

    expect(scenario.messages[0]).toBe('warn: .claude/agents/loremaster-writer.md exists and was not written by LoreMaster. Replace it?')
    expect(store.files.get('.claude/agents/loremaster-writer.md')).toBe('# Mine\n')
    expect(scenario.messages[1]).toContain('skipped .claude/agents/loremaster-writer.md')
    expect(scenario.buttons[0]).toEqual(['Copy MCP Server Config'])
  })

  it('replaces it when the answer is yes', async () => {
    const store = memory({ '.claude/agents/loremaster-writer.md': '# Mine\n' })
    scenario.quickPicks.push(pick('Claude Code subagent'))
    scenario.answers.push('Replace')

    await createAgentCommand({ engine: engine(scenario), files: store.factory })

    expect(store.files.get('.claude/agents/loremaster-writer.md')).toContain('RULES')
  })

  it('does nothing when no target is picked or the pick is cancelled', async () => {
    const store = memory()
    scenario.quickPicks.push(() => undefined, () => [])

    await createAgentCommand({ engine: engine(scenario), files: store.factory })
    await createAgentCommand({ engine: engine(scenario), files: store.factory })

    expect(scenario.requests).toEqual([])
    expect(store.files.size).toBe(0)
  })

  it('shows the engine\'s reason and writes nothing when it cannot give instructions', async () => {
    const store = memory()
    scenario.quickPicks.push(pickAll)

    await createAgentCommand({ engine: engine(scenario, 'outputs[0].mermaidMode "x" is not valid'), files: store.factory })

    expect(scenario.messages).toEqual(['error: LoreMaster: outputs[0].mermaidMode "x" is not valid'])
    expect(store.files.size).toBe(0)
  })

  it('asks for a folder first', async () => {
    setFolders(undefined)

    await createAgentCommand({ engine: engine(scenario), files: memory().factory })

    expect(scenario.messages).toEqual(['info: LoreMaster: open a folder to create its documentation agent.'])
  })
})
