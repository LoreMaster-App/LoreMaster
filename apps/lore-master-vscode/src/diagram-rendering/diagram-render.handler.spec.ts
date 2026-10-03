import { HOST_RENDER_DIAGRAM_METHOD } from '../engine-protocol'
import type { EngineClient } from '../engine-process'
import { answerRenderDiagrams, type DiagramRenderer } from './diagram-render.handler'

type RequestHandler = (params: unknown) => Promise<unknown> | unknown

/** Captures the handler registered for a method, so the test can invoke it as the engine
 *  would. */
function fakeEngine (): EngineClient & { invoke: (method: string, params: unknown) => Promise<unknown> } {
  const handlers = new Map<string, RequestHandler>()

  return {
    request:        () => Promise.resolve(undefined as never),
    onNotification: () => ({ dispose () {} }),
    onRequest (method, handler) {
      handlers.set(method, handler)

      return { dispose () { handlers.delete(method) } }
    },
    dispose () {},
    invoke: (method, params) => Promise.resolve(handlers.get(method)?.(params)),
  }
}

function renderer (svg: string): DiagramRenderer & { seen: string[] } {
  const seen: string[] = []

  return {
    seen,
    render: source => {
      seen.push(source)

      return Promise.resolve(svg)
    },
    dispose () {},
  }
}

describe('answerRenderDiagrams', () => {
  it('renders a Mermaid diagram and returns its SVG', async () => {
    const engine = fakeEngine()
    const draw = renderer('<svg>drawn</svg>')
    answerRenderDiagrams({ engine, renderer: draw })

    const result = await engine.invoke(HOST_RENDER_DIAGRAM_METHOD, { language: 'mermaid', source: 'graph TD; A-->B' })

    expect(draw.seen).toEqual(['graph TD; A-->B'])
    expect(result).toEqual({ svg: '<svg>drawn</svg>' })
  })

  it('refuses a non-Mermaid diagram', async () => {
    const engine = fakeEngine()
    answerRenderDiagrams({ engine, renderer: renderer('x') })

    await expect(engine.invoke(HOST_RENDER_DIAGRAM_METHOD, { language: 'plantuml', source: '@startuml' })).rejects.toThrow('plantuml')
  })
})
