import type { EngineRequester } from '../connection-setup'
import type { Space } from '../engine-protocol'
import { pickSpace } from './space-picker.handler'

const engineWith = (spaces: Space[]): EngineRequester => ({ request: () => Promise.resolve({ spaces } as never) })

describe('pickSpace', () => {
  it('lists the session spaces from space/list and returns the chosen one', async () => {
    const spaces: Space[] = [{ id: '1', key: 'ENG', name: 'Engineering' }, { id: '2', key: 'OPS', name: 'Ops' }]
    let offered: Space[] = []

    const chosen = await pickSpace({
      engine:    engineWith(spaces),
      sessionId: 's1',
      ui:        {
        pickSpace: list => {
          offered = list

          return Promise.resolve(list[1])
        },
      },
    })

    expect(offered).toEqual(spaces)
    expect(chosen).toEqual(spaces[1])
  })

  it('returns undefined when the pick is cancelled', async () => {
    const chosen = await pickSpace({
      engine:    engineWith([{ id: '1', key: 'ENG', name: 'Engineering' }]),
      sessionId: 's1',
      ui:        { pickSpace: () => Promise.resolve(undefined) },
    })

    expect(chosen).toBeUndefined()
  })
})
