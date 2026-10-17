import { CONFIG_DESCRIBE_METHOD, type Output, type Settings, type SettingsReadResult } from '../../src/engine-protocol'
import { describedConfig } from './config-description.fixture'

/** What `settings/read` answers for these settings: the names and summaries the engine would make up, simply. */
export function readResultOf (settings: Settings, extra: Partial<SettingsReadResult> = {}): SettingsReadResult {
  return {
    exists:       true,
    firstSync:    false,
    settings,
    displayNames: settings.outputs.map(output => nameOf(output)),
    summaries:    settings.outputs.map(output => summaryOf(output)),
    ...extra,
  }
}

function nameOf (output: Output): string {
  if (output.name !== undefined && output.name !== '') {
    return output.name
  }

  return output.platform === 'confluence' ? `Confluence · ${output.space || 'no space yet'}` : `${output.platform} · ${output.repo || 'this repository'}`
}

function summaryOf (output: Output): string {
  return output.platform === 'confluence' ? output.baseUrl : output.repo || "this repository's origin"
}

/** The engine routes a test needs to describe the configuration: add them to the fake engine's routes. */
export function describeRoutes (): Record<string, () => unknown> {
  return { [CONFIG_DESCRIBE_METHOD]: () => describedConfig() }
}
