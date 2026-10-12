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
 * Leaves files or folders out of the sync, in every storage or in the chosen ones. The whole
 * settings are read and sent back, so nothing else changes; the engine validates them and keeps
 * the author's comments, and a value it refuses throws with its message.
 */
export function excludeFromSync (deps: ChangeSelectionDeps, entries: string[], scope: SelectionScope): Promise<void> {
  return change(deps, settings => entries.reduce((next, entry) => excludeEntry(next, entry, scope), settings))
}

/** Brings files or folders back into the chosen storages, in one settings write. */
export function includeInSync (deps: ChangeSelectionDeps, entries: string[], indexes: number[]): Promise<void> {
  return change(deps, settings => entries.reduce((next, entry) => includeEntry(next, entry, indexes), settings))
}
