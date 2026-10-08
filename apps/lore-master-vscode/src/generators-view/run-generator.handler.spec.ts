import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { GENERATORS_RUN_METHOD, type Generator } from '../engine-protocol'
import type { GeneratorsViewDeps } from './add-generator.handler'
import { GeneratorsViewProvider } from './generators-view.client'
import { runGeneratorCommand } from './run-generator.handler'

const tests: Generator = { type: 'test-results', output: 'docs/tests' }

const window = vscode.window as unknown as Record<string, unknown>
const original = { ...window }

describe('runGeneratorCommand', () => {
  let messages: string[]
  let requests: unknown[]

  const engine = (fail?: string): EngineClient => ({
    request (method: string, params?: unknown) {
      if (method === GENERATORS_RUN_METHOD) {
        requests.push(params)

        return fail ? Promise.reject(new Error(fail)) : Promise.resolve({ runs: [{ index: 0, type: 'test-results', output: 'docs/tests', written: ['a.md'], unchanged: ['b.md'] }] } as never)
      }

      return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs: [], generators: [tests] } } as never)
    },
  }) as unknown as EngineClient

  const deps = (fail?: string): GeneratorsViewDeps => ({
    engine:   engine(fail),
    output:   { appendLine () {}, show () {} } as unknown as vscode.OutputChannel,
    provider: new GeneratorsViewProvider(engine()),
  })

  beforeEach(() => {
    messages = []
    requests = []
    window.showInformationMessage = (message: string) => {
      messages.push(`info: ${message}`)

      return Promise.resolve(undefined)
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

  it('runs just that generator, reports the summary, and shows the result on its row', async () => {
    const view = deps()
    const [row] = await view.provider.getChildren()

    await runGeneratorCommand(view, row)

    expect(requests).toEqual([{ workspaceRoot: '/w', generators: [0] }])
    expect(messages).toEqual(['info: LoreMaster: 1 page written, 1 unchanged, 0 removed by 1 generator.'])
    expect(view.provider.getTreeItem(row).description).toBe('→ docs/tests · 1 written, 1 unchanged, 0 removed')
  })

  it('shows the engine\'s reason when the call fails', async () => {
    await runGeneratorCommand(deps('boom'), { index: 0, generator: tests })

    expect(messages).toEqual(['error: LoreMaster: boom'])
  })

  it('does nothing without a row', async () => {
    await runGeneratorCommand(deps(), undefined)

    expect(requests).toEqual([])
  })
})
