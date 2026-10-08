import { GENERATORS_RUN_METHOD, type Generator, SETTINGS_READ_METHOD, type Settings } from '../engine-protocol'
import { type GeneratorsEngine, listGenerators, runGenerators } from './run-generators.use-case'

function engine (settings: Settings, requests: { method: string; params: unknown }[] = []): GeneratorsEngine {
  return {
    request (method: string, params?: unknown) {
      requests.push({ method, params })
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings } as never)
      }

      return Promise.resolve({ runs: [{ index: 1, type: 'go-docs', output: 'docs/api' }] } as never)
    },
  }
}

const generators: Generator[] = [{ type: 'test-results', output: 'docs/tests' }, { type: 'go-docs', output: 'docs/api' }]

describe('listGenerators', () => {
  it('returns the configured generators in order', async () => {
    expect(await listGenerators(engine({ version: 1, outputs: [], generators }), '/w')).toEqual(generators)
  })

  it('returns an empty list when the settings have none', async () => {
    expect(await listGenerators(engine({ version: 1, outputs: [] }), '/w')).toEqual([])
  })

  it('reads the settings of the given workspace', async () => {
    const requests: { method: string; params: unknown }[] = []
    await listGenerators(engine({ version: 1, outputs: [] }, requests), '/workspace')

    expect(requests).toEqual([{ method: SETTINGS_READ_METHOD, params: { workspaceRoot: '/workspace' } }])
  })
})

describe('runGenerators', () => {
  it('asks the engine to run the chosen generators', async () => {
    const requests: { method: string; params: unknown }[] = []

    const result = await runGenerators(engine({ version: 1, outputs: [] }, requests), '/w', [1])

    expect(requests).toEqual([{ method: GENERATORS_RUN_METHOD, params: { workspaceRoot: '/w', generators: [1] } }])
    expect(result.runs[0].type).toBe('go-docs')
  })

  it('leaves the indexes out to run them all', async () => {
    const requests: { method: string; params: unknown }[] = []
    await runGenerators(engine({ version: 1, outputs: [] }, requests), '/w')

    expect(requests[0].params).toEqual({ workspaceRoot: '/w', generators: undefined })
  })
})
