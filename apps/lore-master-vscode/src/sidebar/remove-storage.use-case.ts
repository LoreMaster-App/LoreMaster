import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type SettingsReadResult } from '../engine-protocol'
import type { StoragesEngine } from './storages-view.client'

/** The outcome of a removal. */
export type RemoveResult = 'removed'

/** Removes the output at `index` from the settings. Removing the last one is allowed: the
 *  workspace is then configured with no storage, and nothing falls back to an older file. */
export async function removeStorage (deps: { engine: StoragesEngine; workspaceRoot: string; index: number }): Promise<RemoveResult> {
  const { engine, workspaceRoot, index } = deps

  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  const outputs: Output[] = read.settings.outputs.filter((_, position) => position !== index)
  await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { ...read.settings, version: read.settings.version || 1, outputs } })

  return 'removed'
}
