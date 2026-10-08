import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { PAGES_PUBLISH_METHOD, SETTINGS_READ_METHOD, WATCH_ROUTE_METHOD } from '../engine-protocol'
import type { SyncCommandDeps } from '../sync-command'
import { TOGGLE_WATCH_COMMAND, WATCH_DEBOUNCE_SETTING, WatchMode } from './watch-mode.handler'

interface Engine {
  requests: { method: string; params: unknown }[]
  outputs:  { platform: string }[]
  fail:     boolean
}

function deps (engine: Engine, lines: string[]): SyncCommandDeps {
  return {
    engine: {
      request (method: string, params?: unknown) {
        engine.requests.push({ method, params })
        if (method === SETTINGS_READ_METHOD) {
          return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs: engine.outputs } } as never)
        }
        if (method === WATCH_ROUTE_METHOD) {
          return Promise.resolve({ everything: false, generators: [], markdown: (params as { changed: string[] }).changed } as never)
        }
        if (method === PAGES_PUBLISH_METHOD) {
          return engine.fail ? Promise.reject(new Error('push rejected')) : Promise.resolve({ changed: true, files: 2, branch: 'gh-pages' } as never)
        }

        return Promise.resolve({} as never)
      },
    } as unknown as EngineClient,
    connections: {} as never,
    targets:     {} as never,
    output:      { appendLine: (line: string) => { lines.push(line) } } as unknown as vscode.OutputChannel,
  }
}

interface StubWatcher {
  pattern:  unknown
  disposed: boolean
  fire (kind: 'change' | 'create' | 'delete', fsPath: string): void
}

interface StubStatusBarItem {
  text:     string
  tooltip:  string | undefined
  command:  string | undefined
  visible:  boolean
  disposed: boolean
}

const stub = vscode as unknown as { fileSystemWatchers: StubWatcher[]; statusBarItems: StubStatusBarItem[]; settings: Map<string, unknown> }
const window = vscode.window as unknown as Record<string, unknown>
const original = { ...window }
const workspace = vscode.workspace as unknown as { workspaceFolders: unknown }
const messages: string[] = []

beforeEach(() => {
  jest.useFakeTimers()
  messages.length = 0
  stub.fileSystemWatchers.length = 0
  stub.statusBarItems.length = 0
  stub.settings.clear()
  workspace.workspaceFolders = [{ uri: { fsPath: '/work/repo' }, name: 'repo' }]
  window.showInformationMessage = (message: string) => {
    messages.push(`info: ${message}`)

    return Promise.resolve(undefined)
  }
  window.showErrorMessage = (message: string) => {
    messages.push(`error: ${message}`)

    return Promise.resolve(undefined)
  }
})

afterEach(() => {
  jest.useRealTimers()
  Object.assign(window, original)
  workspace.workspaceFolders = undefined
})

const githubPages = [{ platform: 'github-pages' }]

/** Lets real file reads finish: fake timers do not move them. */
async function settleFileReads (): Promise<void> {
  const { setImmediate: realSetImmediate } = jest.requireActual<{ setImmediate: (callback: () => void) => unknown }>('node:timers')
  for (let turn = 0; turn < 50; turn++) {
    await new Promise<void>(resolve => { realSetImmediate(resolve) })
    await jest.advanceTimersByTimeAsync(0)
  }
}

describe('WatchMode', () => {
  it('starts watching the folder, shows it in the status bar, and says so', async () => {
    const watch = new WatchMode(deps({ requests: [], outputs: githubPages, fail: false }, []))

    await watch.toggle()

    expect(watch.watching).toBe(true)
    expect(stub.fileSystemWatchers).toHaveLength(1)
    expect(stub.fileSystemWatchers[0].pattern).toMatchObject({ base: '/work/repo', pattern: '**/*' })
    const [item] = stub.statusBarItems
    expect(item).toMatchObject({ visible: true, command: TOGGLE_WATCH_COMMAND })
    expect(item.text).toContain('watching')
    expect(messages[0]).toContain('LoreMaster is watching /work/repo')
  })

  it('syncs only the pages that changed, once, after the quiet time', async () => {
    const engine: Engine = { requests: [], outputs: githubPages, fail: false }
    const watch = new WatchMode(deps(engine, []))
    await watch.toggle()
    const [watcher] = stub.fileSystemWatchers

    watcher.fire('change', '/work/repo/docs/guide.md')
    await jest.advanceTimersByTimeAsync(1000)
    watcher.fire('create', '/work/repo/README.md')
    await jest.advanceTimersByTimeAsync(1999)
    expect(engine.requests.map(request => request.method)).toEqual([SETTINGS_READ_METHOD])
    await jest.advanceTimersByTimeAsync(1)

    const routed = engine.requests.filter(request => request.method === WATCH_ROUTE_METHOD)
    expect(routed).toHaveLength(1)
    expect(routed[0].params).toEqual({ workspaceRoot: '/work/repo', changed: ['README.md', 'docs/guide.md'] })
    expect(engine.requests.some(request => request.method === PAGES_PUBLISH_METHOD)).toBe(true)
  })

  it('ignores files that cannot matter and files outside the folder', async () => {
    const engine: Engine = { requests: [], outputs: githubPages, fail: false }
    const watch = new WatchMode(deps(engine, []))
    await watch.toggle()
    const [watcher] = stub.fileSystemWatchers

    watcher.fire('change', '/work/repo/node_modules/x/index.js')
    watcher.fire('change', '/work/repo/logo.png')
    watcher.fire('change', '/work/repo/.git/config')
    watcher.fire('change', '/work/elsewhere/a.md')
    await jest.advanceTimersByTimeAsync(10_000)

    expect(engine.requests.map(request => request.method)).toEqual([SETTINGS_READ_METHOD])
  })

  it('uses the debounce the setting asks for', async () => {
    stub.settings.set(`loreMaster.${WATCH_DEBOUNCE_SETTING}`, 5)
    const engine: Engine = { requests: [], outputs: githubPages, fail: false }
    await new WatchMode(deps(engine, [])).toggle()

    stub.fileSystemWatchers[0].fire('change', '/work/repo/README.md')
    await jest.advanceTimersByTimeAsync(4999)
    expect(engine.requests.some(request => request.method === WATCH_ROUTE_METHOD)).toBe(false)
    await jest.advanceTimersByTimeAsync(1)

    expect(engine.requests.some(request => request.method === WATCH_ROUTE_METHOD)).toBe(true)
  })

  it('shows a failed sync in the status bar and tries again', async () => {
    const engine: Engine = { requests: [], outputs: githubPages, fail: true }
    const lines: string[] = []
    await new WatchMode(deps(engine, lines)).toggle()
    const [item] = stub.statusBarItems

    stub.fileSystemWatchers[0].fire('change', '/work/repo/README.md')
    await jest.advanceTimersByTimeAsync(2000)

    expect(item.text).toContain('retrying')
    expect(item.tooltip).toBe('push rejected')
    expect(lines.some(line => line.includes('failed: push rejected; trying again in 2s'))).toBe(true)
    engine.fail = false
    await jest.advanceTimersByTimeAsync(2000)
    await settleFileReads()
    expect(item.text).toContain('watching')
  })

  it('stops when toggled again, releasing the watcher and the status bar item', async () => {
    const engine: Engine = { requests: [], outputs: githubPages, fail: false }
    const watch = new WatchMode(deps(engine, []))
    await watch.toggle()
    const [watcher] = stub.fileSystemWatchers
    const [item] = stub.statusBarItems

    await watch.toggle()
    watcher.fire('change', '/work/repo/README.md')
    await jest.advanceTimersByTimeAsync(10_000)

    expect(watch.watching).toBe(false)
    expect(watcher.disposed).toBe(true)
    expect(item.disposed).toBe(true)
    expect(engine.requests.some(request => request.method === WATCH_ROUTE_METHOD)).toBe(false)
    expect(messages.at(-1)).toBe('info: LoreMaster: stopped watching.')
  })

  it('does not start without a storage to send the pages to', async () => {
    const watch = new WatchMode(deps({ requests: [], outputs: [], fail: false }, []))

    await watch.toggle()

    expect(watch.watching).toBe(false)
    expect(stub.fileSystemWatchers).toHaveLength(0)
    expect(messages[0]).toContain('set up a storage first')
  })

  it('does not start without an open folder', async () => {
    workspace.workspaceFolders = undefined
    const watch = new WatchMode(deps({ requests: [], outputs: githubPages, fail: false }, []))

    await watch.toggle()

    expect(watch.watching).toBe(false)
    expect(messages[0]).toContain('open a folder')
  })

  it('stops when disposed', async () => {
    const watch = new WatchMode(deps({ requests: [], outputs: githubPages, fail: false }, []))
    await watch.toggle()

    watch.dispose()

    expect(stub.fileSystemWatchers[0].disposed).toBe(true)
  })
})
