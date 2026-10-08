import type { EngineClient } from '../engine-process'
import { type Output, SESSION_CLOSE_METHOD, SESSION_OPEN_METHOD, SYNC_PLAN_METHOD, type SyncPlanResult } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import { checkRemote } from './check-remote.use-case'

const output = { platform: 'confluence', baseUrl: 'https://x.atlassian.net/wiki', space: 'ENG' } as Output
const meta = { baseUrl: 'https://X.atlassian.net/wiki/', edition: 'cloud', displayName: 'X' } as unknown as ConnectionMeta

const TOKEN = { kind: 'token' }

function connections (list: ConnectionMeta[], credential: unknown = TOKEN): ConnectionStore {
  return {
    list:       () => list,
    credential: () => Promise.resolve((credential ?? undefined) as never),
    add:        () => Promise.resolve(),
    remove:     () => Promise.resolve(),
  }
}

function engine (handlers: Record<string, (params: unknown) => unknown>, calls: string[] = []): EngineClient {
  return {
    async request (method: string, params?: unknown) {
      calls.push(method)

      return handlers[method](params) as never
    },
    onNotification: () => ({ dispose () {} }),
    dispose () {},
  } as unknown as EngineClient
}

const plan: SyncPlanResult = {
  planId:  'p1',
  counts:  {},
  actions: [
    { kind: 'unchanged', path: 'README.md', title: 'Home' },
    { kind: 'conflict', path: 'docs/a.md', title: 'A' },
    { kind: 'orphan', title: 'Old page', pageId: '9', url: 'https://x/9' },
  ],
}

describe('checkRemote', () => {
  it('opens a session, plans, closes the session and returns the per-file actions and orphans', async () => {
    const calls: string[] = []
    const result = await checkRemote(
      { engine: engine({ [SESSION_OPEN_METHOD]: () => ({ sessionId: 's1' }), [SYNC_PLAN_METHOD]: () => plan, [SESSION_CLOSE_METHOD]: () => null }, calls), connections: connections([meta]), workspaceRoot: '/w' },
      output,
      2,
    )

    expect(result).toEqual({
      ok:    true,
      check: { actions: { 'README.md': 'unchanged', 'docs/a.md': 'conflict' }, orphans: [{ title: 'Old page', pageId: '9', url: 'https://x/9' }] },
    })
    expect(calls).toEqual([SESSION_OPEN_METHOD, SYNC_PLAN_METHOD, SESSION_CLOSE_METHOD])
  })

  it('plans the output it was asked about, with no scope', async () => {
    let params: unknown
    await checkRemote(
      {
        engine: engine({
          [SESSION_OPEN_METHOD]: () => ({ sessionId: 's1' }),
          [SYNC_PLAN_METHOD]:    p => {
            params = p

            return plan
          },
          [SESSION_CLOSE_METHOD]: () => null,
        }),
        connections:   connections([meta]),
        workspaceRoot: '/w',
      },
      output,
      2,
    )

    expect(params).toEqual({ sessionId: 's1', workspaceRoot: '/w', output: 2 })
  })

  it('says why when there is no connection for the site', async () => {
    const result = await checkRemote({ engine: engine({}), connections: connections([]), workspaceRoot: '/w' }, output, 0)

    expect(result.ok).toBe(false)
    expect(!result.ok && result.reason).toContain('No connection for https://x.atlassian.net/wiki')
  })

  it('says why when the credential is gone', async () => {
    const result = await checkRemote({ engine: engine({}), connections: connections([meta], null), workspaceRoot: '/w' }, output, 0)

    expect(!result.ok && result.reason).toContain('No stored credential')
  })

  it('reports a failed session and does not try to plan', async () => {
    const calls: string[] = []
    const result = await checkRemote(
      { engine: engine({ [SESSION_OPEN_METHOD]: () => { throw new Error('unauthorized') } }, calls), connections: connections([meta]), workspaceRoot: '/w' },
      output,
      0,
    )

    expect(result).toEqual({ ok: false, reason: 'unauthorized' })
    expect(calls).toEqual([SESSION_OPEN_METHOD])
  })

  it('closes the session even when planning fails', async () => {
    const calls: string[] = []
    const result = await checkRemote(
      { engine: engine({ [SESSION_OPEN_METHOD]: () => ({ sessionId: 's1' }), [SYNC_PLAN_METHOD]: () => { throw new Error('boom') }, [SESSION_CLOSE_METHOD]: () => null }, calls), connections: connections([meta]), workspaceRoot: '/w' },
      output,
      0,
    )

    expect(result).toEqual({ ok: false, reason: 'boom' })
    expect(calls).toEqual([SESSION_OPEN_METHOD, SYNC_PLAN_METHOD, SESSION_CLOSE_METHOD])
  })
})
