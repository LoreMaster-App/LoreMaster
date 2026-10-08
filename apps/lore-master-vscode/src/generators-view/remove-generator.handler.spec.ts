import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { type Generator, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type Settings } from '../engine-protocol'
import type { GeneratorsViewDeps } from './add-generator.handler'
import { GeneratorsViewProvider } from './generators-view.client'
import { removeGeneratorCommand } from './remove-generator.handler'

const tests: Generator = { type: 'test-results', output: 'docs/tests' }
const api: Generator = { type: 'go-docs', output: 'docs/api' }

const window = vscode.window as unknown as Record<string, unknown>
const original = { ...window }

describe('removeGeneratorCommand', () => {
  let messages: string[]
  let answers: (string | undefined)[]
  let saves: Settings[]

  const engine = (refuse?: string): EngineClient => ({
    request (method: string, params?: unknown) {
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs: [], generators: [tests, api] } } as never)
      }
      if (method === SETTINGS_SAVE_METHOD) {
        if (refuse) {
          return Promise.reject(new Error(refuse))
        }
        saves.push((params as { settings: Settings }).settings)
      }

      return Promise.resolve(null as never)
    },
  }) as unknown as EngineClient

  const deps = (refuse?: string): GeneratorsViewDeps => ({
    engine:   engine(refuse),
    output:   { appendLine () {}, show () {} } as unknown as vscode.OutputChannel,
    provider: new GeneratorsViewProvider(engine()),
  })

  beforeEach(() => {
    messages = []
    answers = []
    saves = []
    window.showWarningMessage = (message: string) => {
      messages.push(`warn: ${message}`)

      return Promise.resolve(answers.shift())
    }
    window.showErrorMessage = (message: string) => {
      messages.push(`error: ${message}`)

      return Promise.resolve(undefined)
    };
    (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = [{ uri: { fsPath: '/w' }, name: 'w' }]
  })
  afterEach(() => {
    Object.assign(window, original);
    (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = undefined
  })

  it('confirms, says the pages stay, and removes the generator', async () => {
    answers.push('Remove')

    await removeGeneratorCommand(deps(), { index: 0, generator: tests })

    expect(messages).toEqual(['warn: Remove the Test results generator? The pages it wrote to docs/tests stay; delete that folder to remove them.'])
    expect(saves[0].generators).toEqual([api])
  })

  it('leaves it when the confirmation is declined, and does nothing without a row', async () => {
    await removeGeneratorCommand(deps(), { index: 0, generator: tests })
    await removeGeneratorCommand(deps(), undefined)

    expect(saves).toEqual([])
  })

  it('shows the engine\'s reason when the save fails', async () => {
    answers.push('Remove')

    await removeGeneratorCommand(deps('settings are invalid'), { index: 1, generator: api })

    expect(messages.at(-1)).toBe('error: LoreMaster: settings are invalid')
  })
})
