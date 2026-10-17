import { DIAGRAM_RENDER_METHOD } from '../engine-protocol'
import { D2Drawings } from './d2-drawings.store'
import type { DrawingEngine } from './markdown-diagrams.contract'

function engine (answer: (source: string) => Promise<{ svg: string }>): DrawingEngine & { asked: unknown[] } {
  const asked: unknown[] = []

  return {
    asked,
    request<R> (method: string, params?: unknown): Promise<R> {
      expect(method).toBe(DIAGRAM_RENDER_METHOD)
      asked.push(params)

      return answer((params as { source: string }).source) as Promise<R>
    },
  }
}

const settle = (): Promise<void> => new Promise(resolve => setImmediate(resolve))

describe('D2Drawings', () => {
  it('reports a new diagram as being drawn, asks the engine once, and tells when it is drawn', async () => {
    const fake = engine(async source => ({ svg: `<svg>${source}</svg>` }))
    const drawn = jest.fn()
    const drawings = new D2Drawings(fake, drawn)

    expect(drawings.drawingOf('a -> b')).toEqual({ state: 'drawing' })
    expect(drawings.drawingOf('a -> b')).toEqual({ state: 'drawing' })
    await settle()

    expect(fake.asked).toEqual([{ language: 'd2', source: 'a -> b' }])
    expect(drawn).toHaveBeenCalledTimes(1)
    expect(drawings.drawingOf('a -> b')).toEqual({ state: 'drawn', svg: '<svg>a -> b</svg>' })
    expect(fake.asked).toHaveLength(1)
  })

  it('keeps the engine\'s reason when a diagram is refused, and does not ask again', async () => {
    const fake = engine(() => Promise.reject(new Error('1:1 unexpected token')))
    const drawn = jest.fn()
    const drawings = new D2Drawings(fake, drawn)

    drawings.drawingOf('a -> {')
    await settle()

    expect(drawings.drawingOf('a -> {')).toEqual({ state: 'failed', message: '1:1 unexpected token' })
    expect(fake.asked).toHaveLength(1)
    expect(drawn).toHaveBeenCalledTimes(1)
  })

  it('forgets the oldest diagrams past its limit', async () => {
    const fake = engine(async source => ({ svg: source }))
    const drawings = new D2Drawings(fake, () => undefined, 2)

    for (const source of ['one', 'two', 'three']) {
      drawings.drawingOf(source)
      await settle()
    }

    expect(drawings.drawingOf('three')).toEqual({ state: 'drawn', svg: 'three' })
    expect(drawings.drawingOf('one')).toEqual({ state: 'drawing' })
  })
})
