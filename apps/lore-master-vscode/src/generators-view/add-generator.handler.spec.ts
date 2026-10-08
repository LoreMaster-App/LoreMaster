import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { GENERATORS_RUN_METHOD, type Generator, type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type Settings } from '../engine-protocol'
import { addGeneratorCommand } from './add-generator.handler'
import { GeneratorsViewProvider } from './generators-view.client'

const storage = (roots: string[]): Output => ({
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots, template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
})

interface Scenario {
  messages:   string[]
  buttons:    string[][]
  answers:    (string | undefined)[]
  quickPicks: ((items: { label: string }[]) => unknown)[]
  inputs:     (string | undefined)[]
  prompts:    { title?: string; value?: string; validateInput?: (value: string) => string | undefined }[]
  saves:      Settings[]
  runs:       unknown[]
}

const window = vscode.window as unknown as Record<string, unknown>
const original = { ...window }

function script (scenario: Scenario): void {
  window.showInformationMessage = (message: string, ...buttons: string[]) => {
    scenario.messages.push(`info: ${message}`); scenario.buttons.push(buttons)

    return Promise.resolve(scenario.answers.shift())
  }
  window.showWarningMessage = (message: string) => {
    scenario.messages.push(`warn: ${message}`)

    return Promise.resolve(undefined)
  }
  window.showErrorMessage = (message: string) => {
    scenario.messages.push(`error: ${message}`)

    return Promise.resolve(undefined)
  }
  window.showQuickPick = (items: { label: string }[]) => Promise.resolve(scenario.quickPicks.shift()?.(items))
  window.showInputBox = (options: { title?: string; value?: string; validateInput?: (value: string) => string | undefined }) => {
    scenario.prompts.push(options)

    return Promise.resolve(scenario.inputs.shift())
  }
}

function engine (scenario: Scenario, settings: Settings, refuse?: string): EngineClient {
  return {
    request (method: string, params?: unknown) {
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings } as never)
      }
      if (method === SETTINGS_SAVE_METHOD) {
        if (refuse) {
          return Promise.reject(new Error(refuse))
        }
        scenario.saves.push((params as { settings: Settings }).settings)
      }
      if (method === GENERATORS_RUN_METHOD) {
        scenario.runs.push(params)

        return Promise.resolve({ runs: [{ index: 0, type: 'go-docs', output: 'docs/api', written: ['a.md'] }] } as never)
      }

      return Promise.resolve(null as never)
    },
  } as unknown as EngineClient
}

const output = { appendLine () {}, show () {} } as unknown as vscode.OutputChannel
const pick = (label: string) => (items: { label: string }[]): unknown => items.find(item => item.label === label)

describe('addGeneratorCommand', () => {
  let scenario: Scenario
  beforeEach(() => {
    scenario = { messages: [], buttons: [], answers: [], quickPicks: [], inputs: [], prompts: [], saves: [], runs: [] }
    script(scenario);
    (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = [{ uri: { fsPath: '/w' }, name: 'w' }]
  })
  afterEach(() => {
    Object.assign(window, original);
    (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = undefined
  })

  const deps = (settings: Settings, refuse?: string) => ({ engine: engine(scenario, settings, refuse), output, provider: new GeneratorsViewProvider(engine(scenario, settings)) })
  const saved = (): Generator[] => scenario.saves[0].generators ?? []

  it('asks the kind, the folder and what to read, and saves the generator', async () => {
    scenario.quickPicks.push(pick('Go package docs'))
    scenario.inputs.push('docs/api', 'libs/, apps/, !**/internal/')

    await addGeneratorCommand(deps({ version: 1, outputs: [storage(['docs'])] }))

    expect(saved()).toEqual([{ type: 'go-docs', output: 'docs/api', input: ['libs/', 'apps/', '!**/internal/'] }])
    expect(scenario.messages).toEqual(['info: LoreMaster: added the Go package docs generator → docs/api.'])
    expect(scenario.buttons).toEqual([['Run now']])
  })

  it('suggests the kind\'s folder, a free one when it is taken, and validates what is typed', async () => {
    scenario.quickPicks.push(pick('Test results'))
    scenario.inputs.push(undefined)

    await addGeneratorCommand(deps({ version: 1, outputs: [storage(['docs'])], generators: [{ type: 'test-results', output: 'docs/tests' }] }))

    expect(scenario.prompts[0].value).toBe('docs/tests-2')
    expect(scenario.prompts[0].validateInput?.('.')).toContain('Name a folder')
    expect(scenario.prompts[0].validateInput?.('docs/x')).toBeUndefined()
    expect(scenario.saves).toEqual([])
  })

  it('leaves the input out when nothing is typed, so the kind\'s default applies', async () => {
    scenario.quickPicks.push(pick('OpenAPI docs'))
    scenario.inputs.push('docs/openapi', ' '.repeat(3))

    await addGeneratorCommand(deps({ version: 1, outputs: [storage(['docs'])] }))

    expect(saved()).toEqual([{ type: 'openapi-docs', output: 'docs/openapi' }])
  })

  it('warns when the folder is outside every folder a storage syncs', async () => {
    scenario.quickPicks.push(pick('Test results'))
    scenario.inputs.push('reports/pages', '')

    await addGeneratorCommand(deps({ version: 1, outputs: [storage(['docs', 'guides'])] }))

    expect(scenario.messages[0]).toContain('reports/pages is not inside a folder any storage syncs (docs, guides)')
    expect(scenario.messages[0]).toContain('Edit storage')
  })

  it('does not warn when a storage reads the folder', async () => {
    scenario.quickPicks.push(pick('Test results'))
    scenario.inputs.push('docs/tests', '')

    await addGeneratorCommand(deps({ version: 1, outputs: [storage(['.'])] }))

    expect(scenario.messages[0]).not.toContain('not inside')
  })

  it('runs the new generator when asked to, and shows the result on its row', async () => {
    scenario.quickPicks.push(pick('Go package docs'))
    scenario.inputs.push('docs/api', '')
    scenario.answers.push('Run now')

    const settings: Settings = { version: 1, outputs: [storage(['docs'])], generators: [{ type: 'test-results', output: 'docs/tests' }] }
    const view = deps(settings)
    await addGeneratorCommand(view)

    expect(scenario.runs).toEqual([{ workspaceRoot: '/w', generators: [1] }])
    expect(scenario.messages.at(-1)).toBe('info: LoreMaster: 1 page written, 0 unchanged, 0 removed by 1 generator.')
  })

  it('does nothing when any prompt is cancelled', async () => {
    scenario.quickPicks.push(() => undefined)
    await addGeneratorCommand(deps({ version: 1, outputs: [] }))

    scenario.quickPicks.push(pick('Test results'))
    scenario.inputs.push(undefined)
    await addGeneratorCommand(deps({ version: 1, outputs: [] }))

    scenario.quickPicks.push(pick('Test results'))
    scenario.inputs.push('docs/tests', undefined)
    await addGeneratorCommand(deps({ version: 1, outputs: [] }))

    expect(scenario.saves).toEqual([])
    expect(scenario.messages).toEqual([])
  })

  it('shows the engine\'s reason when it refuses the generator', async () => {
    scenario.quickPicks.push(pick('Test results'))
    scenario.inputs.push('docs/tests', '')

    await addGeneratorCommand(deps({ version: 1, outputs: [] }, 'generators[0].output overlaps'))

    expect(scenario.messages).toEqual(['error: LoreMaster: generators[0].output overlaps'])
  })

  it('asks for a folder first', async () => {
    (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = undefined

    await addGeneratorCommand(deps({ version: 1, outputs: [] }))

    expect(scenario.messages).toEqual(['info: LoreMaster: open a folder to add a generator.'])
  })
})
