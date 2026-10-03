import { join } from 'node:path'
import { resolveEngineBinary } from './engine-binary.policy'

describe('resolveEngineBinary', () => {
  it('resolves the bundled binary under the extension bin/', () => {
    const path = resolveEngineBinary({ platform: 'linux', extensionPath: '/ext' })

    expect(path).toBe(join('/ext', 'bin', 'lore-master-engine'))
  })

  it('names the binary .exe on win32', () => {
    const path = resolveEngineBinary({ platform: 'win32', extensionPath: '/ext' })

    expect(path.endsWith('lore-master-engine.exe')).toBe(true)
  })

  it('does not name it .exe off win32', () => {
    const path = resolveEngineBinary({ platform: 'darwin', extensionPath: '/ext' })

    expect(path.endsWith('lore-master-engine')).toBe(true)
    expect(path.endsWith('.exe')).toBe(false)
  })

  it('prefers a configured dev path over the bundled binary', () => {
    const path = resolveEngineBinary({ platform: 'linux', extensionPath: '/ext', configuredPath: '/built/engine' })

    expect(path).toBe('/built/engine')
  })

  it('treats a blank configured path as unset', () => {
    const path = resolveEngineBinary({ platform: 'linux', extensionPath: '/ext', configuredPath: ' '.repeat(3) })

    expect(path).toBe(join('/ext', 'bin', 'lore-master-engine'))
  })
})
