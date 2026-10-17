import { codiconFor, describeConfig, platformOf } from '../config-description'
import { type ConfigDescription, type Output, SETTINGS_READ_METHOD, type SettingsReadResult } from '../engine-protocol'

/** One configured output as a row shows it, with its index in the settings' outputs list. */
export interface ConfiguredOutput {
  index:  number
  output: Output
  /** What the engine calls it: its name, or one made up from where it goes. */
  label:  string
  /** Where it goes: the site address, the repository. */
  detail: string
  /** The codicon of its platform. */
  icon:   string
}

/** A storage row of the Sync view, as a menu command is handed it. */
export type StorageNode = ConfiguredOutput

/** The minimal engine surface reading the outputs needs. */
export interface OutputsEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/**
 * An output shows in the lists once it is configured: every setting its platform's description
 * requires is filled (the blank scaffold of a fresh workspace is hidden). One the description does not
 * know is shown, so a project written for a newer LoreMaster is not hidden from its user.
 */
export function isConfiguredOutput (description: ConfigDescription, output: Output): boolean {
  const platform = platformOf(description, output)
  if (!platform) {
    return true
  }
  const held = output as unknown as Record<string, unknown>

  return platform.fields.every(field => field.required !== true || (typeof held[field.key] === 'string' && held[field.key] !== ''))
}

/** The configured outputs of a read of the settings, with the labels and icons the engine's description gives them. */
export function configuredOutputsOf (read: SettingsReadResult, description: ConfigDescription): ConfiguredOutput[] {
  return read.settings.outputs
    .map((output, index): ConfiguredOutput => ({
      index,
      output,
      label:  read.displayNames[index] ?? output.platform,
      detail: read.summaries[index] ?? '',
      icon:   codiconFor(platformOf(description, output)?.icon ?? ''),
    }))
    .filter(each => isConfiguredOutput(description, each.output))
}

/** Reads a folder's settings and the engine's description and returns what is configured. */
export async function readConfiguredOutputs (engine: OutputsEngine, workspaceRoot: string): Promise<{ outputs: ConfiguredOutput[], description: ConfigDescription }> {
  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  const description = await describeConfig(engine)

  return { outputs: configuredOutputsOf(read, description), description }
}
