import { type CheckCache, checkDiagrams } from './check-diagrams.use-case'
import type { DiagramFence } from './diagram-fence.contract'

const d2: DiagramFence = { language: 'd2', source: 'a -> {', firstLine: 4 }
const mermaid: DiagramFence = { language: 'mermaid', source: 'graph TD\n  ???', firstLine: 10 }

type Answers = Record<string, { problems: { line?: number, column?: number, message: string }[], unavailable?: string }>

function engine (answers: Answers): { request: jest.Mock } {
  return {
    request: jest.fn((_method: string, params: { diagrams: { language: string }[] }) => Promise.resolve({ results: params.diagrams.map(each => answers[each.language]) })),
  }
}

describe('checkDiagrams', () => {
  it('places each problem on the line of the document its source line is', async () => {
    const answers = { d2: { problems: [{ line: 1, column: 6, message: 'unclosed map' }] }, mermaid: { problems: [{ line: 2, message: 'Parse error on line 2' }] } }

    const { findings } = await checkDiagrams(engine(answers), [d2, mermaid], new Map(), ['d2', 'mermaid'])

    expect(findings).toEqual([
      { language: 'd2', line: 4, column: 5, message: 'unclosed map' },
      { language: 'mermaid', line: 11, column: 0, message: 'Parse error on line 2' },
    ])
  })

  it('puts a problem with no line on the opening fence, and holds one past the end to the last line', async () => {
    const answers = { d2: { problems: [{ message: 'no line' }, { line: 99, message: 'past the end' }] } }

    const { findings } = await checkDiagrams(engine(answers), [d2], new Map(), ['d2'])

    expect(findings.map(each => each.line)).toEqual([3, 4])
  })

  it('does not ask again about a diagram it knows, and leaves a language it was not asked to check alone', async () => {
    const cache: CheckCache = new Map()
    const asked = engine({ d2: { problems: [] } })
    await checkDiagrams(asked, [d2, mermaid], cache, ['d2'])
    await checkDiagrams(asked, [d2, mermaid], cache, ['d2'])

    expect(asked.request).toHaveBeenCalledTimes(1)
    expect(asked.request.mock.calls[0][1]).toEqual({ diagrams: [{ language: 'd2', source: 'a -> {' }] })
  })

  it('reports why a diagram could not be checked, and tries it again next time', async () => {
    const cache: CheckCache = new Map()
    const asked = engine({ mermaid: { problems: [], unavailable: 'no browser here' } })

    const first = await checkDiagrams(asked, [mermaid], cache, ['mermaid'])
    await checkDiagrams(asked, [mermaid], cache, ['mermaid'])

    expect(first.unavailable).toEqual(['no browser here'])
    expect(asked.request).toHaveBeenCalledTimes(2)
  })

  it('asks once for the same diagram written twice', async () => {
    const asked = engine({ d2: { problems: [{ line: 1, message: 'bad' }] } })

    const { findings } = await checkDiagrams(asked, [d2, { ...d2, firstLine: 20 }], new Map(), ['d2'])

    expect(asked.request.mock.calls[0][1].diagrams).toHaveLength(1)
    expect(findings.map(each => each.line)).toEqual([4, 20])
  })
})
