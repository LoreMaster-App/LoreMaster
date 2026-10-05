import type { EngineClient } from '../engine-process'
import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD } from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import type { StorageSetupUI, StorageType } from '../storage-setup'
import { addStorage } from './add-storage.use-case'

type Route = (params: unknown) => unknown

interface FakeEngine extends EngineClient {
  calls: { method: string; params: unknown }[]
}

function fakeEngine (routes: Record<string, Route>): FakeEngine {
  const calls: { method: string; params: unknown }[] = []

  return {
    calls,
    request (method, params) {
      calls.push({ method, params })
      const route: Route | undefined = routes[method]

      return Promise.resolve((route ? route(params) : null) as never)
    },
    onNotification () {
      return { dispose () {} }
    },
    onRequest () {
      return { dispose () {} }
    },
    dispose () {},
  }
}

function connectionStore (): ConnectionStore {
  return {
    list:       () => [],
    credential: () => Promise.resolve(undefined),
    add:        () => Promise.resolve(),
    remove:     () => Promise.resolve(),
  }
}

function buildUI (types: StorageType[] | undefined): StorageSetupUI & { report (lines: string[]): void; reported: string[][]; errors: string[] } {
  const reported: string[][] = []
  const errors: string[] = []

  return {
    reported,
    errors,
    pickStorageTypes:  () => Promise.resolve(types),
    pickSpace:         spaces => Promise.resolve(spaces[0]),
    pickAction:        () => Promise.resolve({ kind: 'use' }),
    promptSearchQuery: () => Promise.resolve(undefined),
    pickSearchResult:  () => Promise.resolve(undefined),
    promptTitlePrefix: value => Promise.resolve(value),
    pickConnection:    metas => Promise.resolve(metas[0]),
    noConnections:     () => Promise.resolve(),
    promptRepo:        () => Promise.resolve(''),
    promptBranch:      () => Promise.resolve('gh-pages'),
    report:            lines => { reported.push(lines) },
    error:             message => {
      errors.push(message)

      return Promise.resolve()
    },
  }
}

const pagesOutput: Output = {
  platform:       'github-pages',
  baseUrl:        '',
  space:          '',
  parentPageId:   '',
  titlePrefix:    '',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    '',
  titleCollision: '',
  linkMode:       '',
  repo:           '',
  branch:         'gh-pages',
}

const confluenceOutput: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}

describe('addStorage', () => {
  it('appends a new output beside the existing ones', async () => {
    let saved: unknown
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [confluenceOutput] } }),
      [SETTINGS_SAVE_METHOD]: params => {
        saved = params

        return null
      },
    })
    const ui = buildUI(['github-pages'])

    await addStorage({ engine, connections: connectionStore(), workspaceRoot: '/w', ui })

    expect((saved as { settings: { outputs: Output[] } }).settings.outputs).toEqual([confluenceOutput, pagesOutput])
    expect(ui.reported[0][0]).toContain('Added 1 storage')
  })

  it('replaces the scaffold on a fresh workspace', async () => {
    let saved: unknown
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: false, firstSync: true, settings: { version: 1, outputs: [{ ...confluenceOutput, baseUrl: '', space: '', parentPageId: '', titlePrefix: '' }] } }),
      [SETTINGS_SAVE_METHOD]: params => {
        saved = params

        return null
      },
    })

    await addStorage({ engine, connections: connectionStore(), workspaceRoot: '/w', ui: buildUI(['github-pages']) })

    expect((saved as { settings: { outputs: Output[] } }).settings.outputs).toEqual([pagesOutput])
  })

  it('saves nothing when the user cancels the type picker', async () => {
    const engine = fakeEngine({
      [SETTINGS_READ_METHOD]: () => ({ exists: true, firstSync: false, settings: { version: 1, outputs: [confluenceOutput] } }),
    })

    await addStorage({ engine, connections: connectionStore(), workspaceRoot: '/w', ui: buildUI(undefined) })

    expect(engine.calls.some(call => call.method === SETTINGS_SAVE_METHOD)).toBe(false)
  })
})
