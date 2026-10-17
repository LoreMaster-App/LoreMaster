import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import type { ConfigDescription } from '../../src/engine-protocol'

/**
 * What the engine's `config/describe` answers, as a test can use it. The JSON beside it is written
 * from the real catalog (a Go test keeps it current), so the extension is tested against the engine's
 * own description and not one made up here. Each call returns a fresh copy.
 */
export function describedConfig (): ConfigDescription {
  return JSON.parse(readFileSync(join(__dirname, 'config-description.fixture.json'), 'utf8')) as ConfigDescription
}
