import { SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type SettingsReadResult } from '../engine-protocol'
import type { StoragesEngine } from '../sidebar'
import type { StorageField } from './storage-field.config'

/**
 * Changes one setting of the storage at `index` in .lore-master.yaml. The whole settings are
 * read and sent back, so nothing else changes; the engine validates them and keeps the
 * author's comments. A value the engine refuses throws with its message, and nothing is
 * written.
 */
export async function editStorage (deps: { engine: StoragesEngine; workspaceRoot: string; index: number }, field: StorageField, value: string): Promise<void> {
  const { engine, workspaceRoot, index } = deps

  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  if (index < 0 || index >= read.settings.outputs.length) {
    throw new Error('That storage is no longer in .lore-master.yaml.')
  }
  const outputs = read.settings.outputs.map((output, position) => position === index ? field.write(output, value) : output)

  await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { ...read.settings, outputs } })
}
