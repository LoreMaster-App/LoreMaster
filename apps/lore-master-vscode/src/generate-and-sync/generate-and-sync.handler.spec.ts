import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { GENERATORS_RUN_METHOD, type Generator, type GeneratorRun, SETTINGS_READ_METHOD } from '../engine-protocol'
import { generateAndSyncCommand, syncCommand } from './generate-and-sync.handler'

interface Scenario {
  messages:   string[]
  answers:    (string | undefined)[]
  requests:   string[]
  syncs:      (string | undefined)[]
  generators: Generator[]
  runs:       GeneratorRun[]
  fail?:      string
}

const window = vscode.window as unknown as Record<string, unknown>
const workspace = vscode.workspace as unknown as { getConfiguration: unknown; workspaceFolders: unknown }
const original = { ...window }
const originalConfiguration = workspace.getConfiguration

function engine (scenario: Scenario): EngineClient {
  return {
    request (method: string) {
      scenario.requests.push(method)
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs: [], generators: scenario.generators } } as never)
      }
      if (method === GENERATORS_RUN_METHOD && scenario.fail) {
        return Promise.reject(new Error(scenario.fail))
      }

      return Promise.resolve({ runs: scenario.runs } as never)
    },
  } as unknown as EngineClient
}

const tests: Generator = { type: 'test-results', output: 'docs/tests' }

function withSetting (value: boolean | undefined): void {
  workspace.getConfiguration = () => ({ get: (_key: string, fallback: unknown) => value ?? fallback })
}

describe('generate and sync', () => {
  let scenario: Scenario
  const deps = () => ({
    engine:      engine(scenario),
    connections: {} as never,
    targets:     {} as never,
    output:      { appendLine () {}, show () {} } as unknown as vscode.OutputChannel,
    runSync:     (_deps: unknown, folder?: string) => {
      scenario.syncs.push(folder)

      return Promise.resolve()
    },
  })

  beforeEach(() => {
    scenario = { messages: [], answers: [], requests: [], syncs: [], generators: [tests], runs: [{ index: 0, type: 'test-results', output: 'docs/tests', written: ['a.md'] }] }
    window.showInformationMessage = (message: string) => {
      scenario.messages.push(`info: ${message}`)

      return Promise.resolve(undefined)
    }
    window.showWarningMessage = (message: string) => {
      scenario.messages.push(`warn: ${message}`)

      return Promise.resolve(scenario.answers.shift())
    }
    window.showErrorMessage = (message: string) => {
      scenario.messages.push(`error: ${message}`)

      return Promise.resolve(undefined)
    }
    workspace.workspaceFolders = [{ uri: { fsPath: '/w' }, name: 'w' }]
  })
  afterEach(() => {
    Object.assign(window, original)
    workspace.getConfiguration = originalConfiguration
    workspace.workspaceFolders = undefined
  })

  describe('generateAndSyncCommand', () => {
    it('runs the generators, then syncs the folder', async () => {
      await generateAndSyncCommand(deps())

      expect(scenario.requests).toEqual([SETTINGS_READ_METHOD, GENERATORS_RUN_METHOD])
      expect(scenario.syncs).toEqual(['/w'])
      expect(scenario.messages).toEqual(['info: LoreMaster: 1 page written, 0 unchanged, 0 removed by 1 generator.'])
    })

    it('syncs on warnings alone, without asking', async () => {
      scenario.runs = [{ index: 0, type: 'test-results', output: 'docs/tests', warnings: ['reports/bad.xml: not valid'] }]

      await generateAndSyncCommand(deps())

      expect(scenario.syncs).toEqual(['/w'])
      expect(scenario.messages.some(message => message.startsWith('warn: LoreMaster: 0 pages written'))).toBe(true)
      expect(scenario.messages.some(message => message.includes('Sync anyway?'))).toBe(false)
    })

    it('asks before syncing when a generator failed, and syncs only when told to', async () => {
      scenario.runs = [{ index: 0, type: 'test-results', output: 'docs/tests', error: 'no reports' }]

      await generateAndSyncCommand(deps())
      expect(scenario.syncs).toEqual([])
      expect(scenario.messages.at(-1)).toBe('warn: The test-results generator failed, so its pages may be missing or out of date. Sync anyway?')

      scenario.answers.push(undefined, 'Sync anyway')
      await generateAndSyncCommand(deps())
      expect(scenario.syncs).toEqual(['/w'])
    })

    it('says so and syncs the Markdown when there are no generators', async () => {
      scenario.generators = []

      await generateAndSyncCommand(deps())

      expect(scenario.requests).toEqual([SETTINGS_READ_METHOD])
      expect(scenario.messages).toEqual(['info: LoreMaster: no generators are configured, so only the Markdown is synced. Add one from the Generators view.'])
      expect(scenario.syncs).toEqual(['/w'])
    })

    it('shows the engine\'s reason and does not sync when the generators cannot be run', async () => {
      scenario.fail = 'settings are invalid'

      await generateAndSyncCommand(deps())

      expect(scenario.messages.at(-1)).toBe('error: LoreMaster: settings are invalid')
      expect(scenario.syncs).toEqual([])
    })

    it('asks for a folder first', async () => {
      workspace.workspaceFolders = undefined

      await generateAndSyncCommand(deps())

      expect(scenario.messages).toEqual(['info: LoreMaster: open a folder to generate and sync its documentation.'])
      expect(scenario.syncs).toEqual([])
    })
  })

  describe('syncCommand', () => {
    it('only syncs by default', async () => {
      withSetting(undefined)

      await syncCommand(deps())

      expect(scenario.requests).toEqual([])
      expect(scenario.syncs).toEqual([undefined])
    })

    it('generates first when the setting says so', async () => {
      withSetting(true)

      await syncCommand(deps())

      expect(scenario.requests).toEqual([SETTINGS_READ_METHOD, GENERATORS_RUN_METHOD])
      expect(scenario.syncs).toEqual(['/w'])
    })
  })
})
