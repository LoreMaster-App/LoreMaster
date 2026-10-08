import * as vscode from 'vscode'
import { createWorkspaceFileStore } from './workspace-file-store.client'

const stub = vscode as unknown as { virtualFiles: Map<string, Uint8Array>; virtualDirectories: Set<string> }

describe('createWorkspaceFileStore', () => {
  beforeEach(() => {
    stub.virtualFiles.clear()
    stub.virtualDirectories.clear()
  })

  it('reads a file under the folder as text, and undefined when it is missing', async () => {
    stub.virtualFiles.set('/w/docs/a.md', new TextEncoder().encode('# A\n'))
    const store = createWorkspaceFileStore('/w')

    expect(await store.read('docs/a.md')).toBe('# A\n')
    expect(await store.read('docs/missing.md')).toBeUndefined()
  })

  it('writes text, creating the folders above the file', async () => {
    const store = createWorkspaceFileStore('/w')

    await store.write('.claude/agents/x.md', 'hello — world\n')

    expect(new TextDecoder().decode(stub.virtualFiles.get('/w/.claude/agents/x.md'))).toBe('hello — world\n')
    expect(stub.virtualDirectories.has('/w/.claude/agents')).toBe(true)
  })

  it('writes a file at the root without creating a folder', async () => {
    await createWorkspaceFileStore('/w').write('AGENTS.md', 'x')

    expect(stub.virtualFiles.has('/w/AGENTS.md')).toBe(true)
    expect(stub.virtualDirectories.size).toBe(0)
  })

  it('does not hide an error that is not a missing file', async () => {
    const failing = vscode.workspace.fs as unknown as { readFile: (uri: unknown) => Promise<Uint8Array> }
    const original = failing.readFile
    failing.readFile = () => Promise.reject(new Error('permission denied'))

    try {
      await expect(createWorkspaceFileStore('/w').read('a.md')).rejects.toThrow('permission denied')
    } finally {
      failing.readFile = original
    }
  })
})
