import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type SettingsReadResult } from '../engine-protocol'
import type { StoragesEngine } from './storages-view.client'

/** The outcome of a removal: it happened, or it was the only storage (which the config
 *  cannot be without, so nothing was written). */
export type RemoveResult = 'removed' | 'last'

/** Removes the output at `index` from .lore-master.yaml. The config must keep at least one
 *  output, so removing the last one is refused (returns 'last') rather than writing an empty,
 *  invalid file. */
export async function removeStorage (deps: { engine: StoragesEngine; workspaceRoot: string; index: number }): Promise<RemoveResult> {
  const { engine, workspaceRoot, index } = deps

  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  const outputs: Output[] = read.settings.outputs.filter((_, position) => position !== index)
  if (outputs.length === 0) {
    return 'last'
  }

  await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { version: read.settings.version || 1, outputs } })

  return 'removed'
}
