import type { EngineClient } from '../engine-process'
import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type SettingsReadResult } from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { setUpStorages, type StorageSetupUI } from '../storage-setup'

/** What adding a storage needs: the engine, the stored connections, the folder, and a UI
 *  that can run the storage setup and report back. */
export interface AddStorageDeps {
  engine:        EngineClient
  connections:   ConnectionStore
  workspaceRoot: string
  ui:            StorageSetupUI & { report (lines: string[]): void }
}

/**
 * Configures one or more new outputs and appends them to .lore-master.yaml. A fresh
 * workspace (only the unconfigured scaffold) is replaced by what is set up; otherwise the
 * new outputs are added beside the existing ones.
 */
export async function addStorage (deps: AddStorageDeps): Promise<void> {
  const { engine, connections, workspaceRoot, ui } = deps

  let read: SettingsReadResult
  try {
    read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  } catch (error) {
    await ui.error(messageOf(error))

    return
  }

  const created = await setUpStorages({ engine, connections, ui })
  if (!created || created.length === 0) {
    return
  }

  const outputs: Output[] = read.firstSync ? created : [...read.settings.outputs, ...created]
  try {
    await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { version: read.settings.version || 1, outputs } })
  } catch (error) {
    await ui.error(messageOf(error))

    return
  }

  ui.report([`Added ${created.length} storage${created.length === 1 ? '' : 's'} to .lore-master.yaml.`])
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
