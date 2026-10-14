import { FILES_PLAN_MOVE_METHOD } from '../engine-protocol'
import { planMove } from './plan-move.use-case'

describe('planMove', () => {
  it('asks the engine for the plan of the moves', async () => {
    const calls: unknown[][] = []
    const engine = {
      request: <T>(method: string, params: unknown): Promise<T> => {
        calls.push([method, params])

        return Promise.resolve({ renames: [], edits: [], warnings: [] } as T)
      },
    }

    const plan = await planMove(engine, '/work/repo', [{ from: 'a.md', to: 'b.md' }])

    expect(plan).toEqual({ renames: [], edits: [], warnings: [] })
    expect(calls).toEqual([[FILES_PLAN_MOVE_METHOD, { workspaceRoot: '/work/repo', moves: [{ from: 'a.md', to: 'b.md' }] }]])
  })

  it('answers undefined when the engine fails, so the move is not held back', async () => {
    const engine = { request: (): Promise<never> => Promise.reject(new Error('engine gone')) }

    expect(await planMove(engine, '/work/repo', [])).toBeUndefined()
  })
})
