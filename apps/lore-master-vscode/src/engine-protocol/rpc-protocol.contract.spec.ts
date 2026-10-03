import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import * as protocol from './rpc-protocol.contract'

// The engine's Go rpcprotocol package is the source of truth. This asserts the TS mirror
// routes the same method names — a renamed or new engine method, or a stale TS constant,
// fails here rather than as a silent "method not found" at runtime (#57). Field shapes are
// not codegen-checked; keep them in step by hand when a contract changes.
const RPCPROTOCOL_DIR = resolve(__dirname, '../../../lore-master-engine/rpcprotocol')

function goMethods (): Set<string> {
  const methods = new Set<string>()
  for (const file of readdirSync(RPCPROTOCOL_DIR)) {
    if (!file.endsWith('.go')) {
      continue
    }
    const source = readFileSync(join(RPCPROTOCOL_DIR, file), 'utf8')
    for (const match of source.matchAll(/const Method\w+ = "([^"]+)"/g)) {
      methods.add(match[1])
    }
  }

  return methods
}

function tsMethods (): Set<string> {
  const methods = new Set<string>()
  for (const [name, value] of Object.entries(protocol)) {
    if (typeof value === 'string' && name.endsWith('_METHOD')) {
      methods.add(value)
    }
  }

  return methods
}

const byName = (a: string, b: string): number => a.localeCompare(b)

describe('the TS protocol mirror against the Go rpcprotocol package', () => {
  it('routes exactly the same method names', () => {
    const go = goMethods()
    const ts = tsMethods()

    expect(go.size).toBeGreaterThan(0)
    expect([...ts].sort(byName)).toEqual([...go].sort(byName))
  })
})
