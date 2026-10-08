import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { type Generator, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type Settings } from '../engine-protocol'
import type { GeneratorsViewDeps } from './add-generator.handler'
import { editGeneratorCommand } from './edit-generator.handler'
import { GeneratorsViewProvider } from './generators-view.client'

const tests: Generator = { type: 'test-results', output: 'docs/tests', input: ['reports/'], title: 'CI results' }
const api: Generator = { type: 'go-docs', output: 'docs/api' }

interface Scenario {
  messages:   string[]
  quickPicks: ((items: { label: string; description: string }[]) => unknown)[]
  inputs:     (string | undefined)[]
  prompts:    { value?: string; validateInput?: (value: string) => string | undefined }[]
  saves:      Settings[]
  picked?:    { label: string; description: string }[]
}

const window = vscode.window as unknown as Record<string, unknown>
const original = { ...window }

function script (scenario: Scenario): void {
  window.showErrorMessage = (message: string) => {
    scenario.messages.push(`error: ${message}`)

    return Promise.resolve(undefined)
  }
  window.showQuickPick = (items: { label: string; description: string }[]) => {
    scenario.picked = items

    return Promise.resolve(scenario.quickPicks.shift()?.(items))
  }
  window.showInputBox = (options: { value?: string; validateInput?: (value: string) => string | undefined }) => {
    scenario.prompts.push(options)

    return Promise.resolve(scenario.inputs.shift())
  }
}

function engine (scenario: Scenario, refuse?: string): EngineClient {
  return {
    request (method: string, params?: unknown) {
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs: [], generators: [tests, api] } } as never)
      }
      if (method === SETTINGS_SAVE_METHOD) {
        if (refuse) {
          return Promise.reject(new Error(refuse))
        }
        scenario.saves.push((params as { settings: Settings }).settings)
      }

      return Promise.resolve(null as never)
    },
  } as unknown as EngineClient
}

const pick = (label: string) => (items: { label: string }[]): unknown => items.find(item => item.label === label)

describe('editGeneratorCommand', () => {
  let scenario: Scenario
  beforeEach(() => {
    scenario = { messages: [], quickPicks: [], inputs: [], prompts: [], saves: [] }
    script(scenario);
    (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = [{ uri: { fsPath: '/w' }, name: 'w' }]
  })
  afterEach(() => {
    Object.assign(window, original);
    (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = undefined
  })

  const deps = (refuse?: string): GeneratorsViewDeps => ({
    engine:   engine(scenario, refuse),
    output:   { appendLine () {}, show () {} } as unknown as vscode.OutputChannel,
    provider: new GeneratorsViewProvider(engine(scenario)),
  })

  it('shows the current values, then changes the output folder', async () => {
    scenario.quickPicks.push(pick('Output folder'))
    scenario.inputs.push(String.raw` docs\ci `)

    await editGeneratorCommand(deps(), { index: 0, generator: tests })

    expect(scenario.picked?.map(item => [item.label, item.description])).toEqual([['Output folder', 'docs/tests'], ['What it reads', 'reports/'], ['Index page title', 'CI results']])
    expect(scenario.prompts[0].value).toBe('docs/tests')
    expect(scenario.prompts[0].validateInput?.('..')).toContain('inside the workspace')
    expect(scenario.saves[0].generators?.[0]).toEqual({ ...tests, output: 'docs/ci' })
    expect(scenario.saves[0].generators?.[1]).toEqual(api)
  })

  it('changes what it reads, and clears it when the box is emptied', async () => {
    scenario.quickPicks.push(pick('What it reads'))
    scenario.inputs.push('a/, b/')
    await editGeneratorCommand(deps(), { index: 0, generator: tests })
    expect(scenario.saves[0].generators?.[0].input).toEqual(['a/', 'b/'])

    scenario.quickPicks.push(pick('What it reads'))
    scenario.inputs.push('')
    await editGeneratorCommand(deps(), { index: 0, generator: tests })
    expect(scenario.saves[1].generators?.[0].input).toBeUndefined()
  })

  it('changes and clears the index page title', async () => {
    scenario.quickPicks.push(pick('Index page title'))
    scenario.inputs.push('  Nightly  ')
    await editGeneratorCommand(deps(), { index: 0, generator: tests })
    expect(scenario.saves[0].generators?.[0].title).toBe('Nightly')

    scenario.quickPicks.push(pick('Index page title'))
    scenario.inputs.push('')
    await editGeneratorCommand(deps(), { index: 0, generator: tests })
    expect(scenario.saves[1].generators?.[0].title).toBeUndefined()
  })

  it('shows the default for a setting that is not set', async () => {
    scenario.quickPicks.push(() => undefined)

    await editGeneratorCommand(deps(), { index: 1, generator: api })

    expect(scenario.picked?.map(item => item.description)).toEqual(['docs/api', '(the default)', '(the default)'])
  })

  it('does nothing when cancelled at either prompt, or without a row', async () => {
    scenario.quickPicks.push(() => undefined)
    await editGeneratorCommand(deps(), { index: 0, generator: tests })
    scenario.quickPicks.push(pick('Output folder'))
    scenario.inputs.push(undefined)
    await editGeneratorCommand(deps(), { index: 0, generator: tests })
    await editGeneratorCommand(deps(), undefined)

    expect(scenario.saves).toEqual([])
    expect(scenario.messages).toEqual([])
  })

  it('shows the engine\'s reason when it refuses the value', async () => {
    scenario.quickPicks.push(pick('Output folder'))
    scenario.inputs.push('docs/api')

    await editGeneratorCommand(deps('generators[0] and generators[1] write into overlapping folders'), { index: 0, generator: tests })

    expect(scenario.messages).toEqual(['error: LoreMaster: generators[0] and generators[1] write into overlapping folders'])
  })
})
