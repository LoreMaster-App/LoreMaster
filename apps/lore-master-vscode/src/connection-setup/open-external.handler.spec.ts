import { HOST_OPEN_EXTERNAL_METHOD } from '../engine-protocol'
import type { EngineClient } from '../engine-process'
import { answerOpenExternal } from './open-external.handler'

type RequestHandler = (params: unknown) => Promise<unknown> | unknown

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

describe('answerOpenExternal', () => {
  it('opens the URL the engine asks for and acknowledges', async () => {
    const engine = fakeEngine()
    const opened: string[] = []
    answerOpenExternal({ engine, open: async url => { opened.push(url) } })

    const result = await engine.invoke(HOST_OPEN_EXTERNAL_METHOD, { url: 'https://dc.example/rest/oauth2/latest/authorize?x=1' })

    expect(opened).toEqual(['https://dc.example/rest/oauth2/latest/authorize?x=1'])
    expect(result).toEqual({})
  })
})
