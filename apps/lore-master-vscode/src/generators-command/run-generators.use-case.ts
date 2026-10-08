import {
  GENERATORS_RUN_METHOD,
  type Generator,
  type GeneratorsRunResult,
  SETTINGS_READ_METHOD,
  type SettingsReadResult,
} from '../engine-protocol'

/** The minimal engine surface the command needs. */
export interface GeneratorsEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** The generators the workspace's .lore-master.yaml configures, in order (an empty list when none). */
export async function listGenerators (engine: GeneratorsEngine, workspaceRoot: string): Promise<Generator[]> {
  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })

  return read.settings.generators ?? []
}

/** Runs the generators at `indexes` (all of them when absent). The engine writes the pages;
 *  a generator that fails is reported in its own entry rather than failing the call. */
export function runGenerators (engine: GeneratorsEngine, workspaceRoot: string, indexes?: number[]): Promise<GeneratorsRunResult> {
  return engine.request<GeneratorsRunResult>(GENERATORS_RUN_METHOD, { workspaceRoot, generators: indexes })
}
