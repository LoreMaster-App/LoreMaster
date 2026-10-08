import type { GeneratorRun } from '../engine-protocol'
import { confirmationAfterFailures, failedRuns } from './generation-failures.policy'

const run = (overrides: Partial<GeneratorRun>): GeneratorRun => ({ index: 0, type: 'test-results', output: 'docs/tests', ...overrides })

describe('failedRuns', () => {
  it('picks the generators that could not run, not the ones that ran with warnings', () => {
    const failed = run({ index: 1, type: 'go-docs', error: 'no packages' })

    expect(failedRuns({ runs: [run({ written: ['a.md'] }), failed, run({ index: 2, warnings: ['skipped x'] }), run({ index: 3, error: '' })] })).toEqual([failed])
  })

  it('is empty when every generator ran', () => {
    expect(failedRuns({ runs: [run({})] })).toEqual([])
    expect(failedRuns({ runs: [] })).toEqual([])
  })
})

describe('confirmationAfterFailures', () => {
  it('names the one generator that failed', () => {
    expect(confirmationAfterFailures([run({ type: 'ts-docs' })])).toBe('The ts-docs generator failed, so its pages may be missing or out of date. Sync anyway?')
  })

  it('counts and names several', () => {
    expect(confirmationAfterFailures([run({ type: 'ts-docs' }), run({ type: 'go-docs' })])).toBe('2 generators failed (ts-docs, go-docs), so their pages may be missing or out of date. Sync anyway?')
  })
})
