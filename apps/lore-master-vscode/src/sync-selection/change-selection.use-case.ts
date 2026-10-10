import { type Settings, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type SettingsReadResult } from '../engine-protocol'
import type { StoragesEngine } from '../sidebar'
import { excludeEntry, includeEntry, type SelectionScope } from './selection-lists.algorithm'

export interface ChangeSelectionDeps {
  engine:        StoragesEngine
  workspaceRoot: string
}

async function change (deps: ChangeSelectionDeps, edit: (settings: Settings) => Settings): Promise<void> {
  const read = await deps.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: deps.workspaceRoot })

  await deps.engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot: deps.workspaceRoot, settings: edit(read.settings) })
}

/**
 * Leaves a file or folder out of the sync, in every storage or in the chosen ones. The whole
 * settings are read and sent back, so nothing else changes; the engine validates them and keeps
 * the author's comments, and a value it refuses throws with its message.
 */
export function excludeFromSync (deps: ChangeSelectionDeps, entry: string, scope: SelectionScope): Promise<void> {
  return change(deps, settings => excludeEntry(settings, entry, scope))
}

/** Brings a file or folder back into the chosen storages. */
export function includeInSync (deps: ChangeSelectionDeps, entry: string, indexes: number[]): Promise<void> {
  return change(deps, settings => includeEntry(settings, entry, indexes))
}
