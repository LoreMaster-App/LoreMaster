import type { GeneratorRun } from '../engine-protocol'
import { summariseRun } from './generator-run-report.mapper'

const run = (overrides: Partial<GeneratorRun>): GeneratorRun => ({ index: 0, type: 'test-results', output: 'docs/tests', ...overrides })

describe('summariseRun', () => {
  it('says there is nothing to run when there are no generators', () => {
    expect(summariseRun({ runs: [] })).toEqual({ headline: 'LoreMaster: there are no generators to run.', lines: [], hasProblems: false })
  })

  it('counts what was written, left alone and removed across generators', () => {
    const report = summariseRun({
      runs: [
        run({ written: ['a.md', 'b.md'], unchanged: ['c.md'] }),
        run({ index: 1, type: 'go-docs', output: 'docs/api', written: ['d.md'], removed: ['e.md'] }),
      ],
    })

    expect(report.headline).toBe('LoreMaster: 3 pages written, 1 unchanged, 1 removed by 2 generators.')
    expect(report.lines).toEqual([
      'test-results → docs/tests: 2 pages written, 1 unchanged, 0 removed',
      'go-docs → docs/api: 1 page written, 0 unchanged, 1 removed',
    ])
    expect(report.hasProblems).toBe(false)
  })

  it('uses the singular for one generator and one page', () => {
    expect(summariseRun({ runs: [run({ written: ['a.md'] })] }).headline).toBe('LoreMaster: 1 page written, 0 unchanged, 0 removed by 1 generator.')
  })

  it('lists warnings under their generator and calls the report a problem report', () => {
    const report = summariseRun({ runs: [run({ unchanged: ['a.md'], warnings: ['reports/bad.xml: not valid JUnit XML'] })] })

    expect(report.lines).toEqual(['test-results → docs/tests: 0 pages written, 1 unchanged, 0 removed', '  ! reports/bad.xml: not valid JUnit XML'])
    expect(report.hasProblems).toBe(true)
  })

  it('shows a generator that failed, and counts it in the headline', () => {
    const report = summariseRun({ runs: [run({ error: 'generator "test-results": no reports found' }), run({ index: 1, written: ['x.md'] })] })

    expect(report.headline).toBe('LoreMaster: 1 page written, 0 unchanged, 0 removed, 1 failed by 2 generators.')
    expect(report.lines.slice(0, 2)).toEqual(['test-results → docs/tests: failed', '  ✗ generator "test-results": no reports found'])
    expect(report.hasProblems).toBe(true)
  })
})
