import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type SettingsReadResult } from '../engine-protocol'
import type { StorageFormValues } from './storage-form.contract'
import { ignoreFromValues, suggestStorageName, valuesToOutput } from './storage-form-values.mapper'
import { type StorageFormErrors, validateStorageForm } from './storage-form.validator'

/** The minimal engine surface saving needs. */
export interface SaveStorageEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

export interface SaveStorageDeps {
  engine:          SaveStorageEngine
  workspaceRoot:   string
  /** Position of the storage being edited; absent when the form creates a new one. */
  index?:          number
  connectionCount: number
}

export type SaveStorageResult =
  | { saved: true } |
  { saved: false, errors: StorageFormErrors, message?: string }

/**
 * Validates the form and writes it to the settings: a new storage is appended (replacing the
 * unconfigured scaffold of a fresh workspace), an edited one replaces its place, and the
 * workspace-wide ignore list is updated. Nothing else in the settings changes; the engine
 * validates the whole and its refusal is returned for the form to show.
 */
export async function saveStorage (deps: SaveStorageDeps, submitted: StorageFormValues): Promise<SaveStorageResult> {
  try {
    const read = await deps.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: deps.workspaceRoot })
    const otherNames = read.settings.outputs
      .filter((_, position) => position !== deps.index)
      .map(output => output.name ?? '')
      .filter(name => name !== '')
    const errors = validateStorageForm(submitted, deps.connectionCount, otherNames)
    if (Object.keys(errors).length > 0) {
      return { saved: false, errors }
    }
    const values = submitted.name.trim() === '' ? { ...submitted, name: suggestStorageName(submitted, otherNames) } : submitted
    const outputs = placeOutput(read, deps.index, values)
    if (typeof outputs === 'string') {
      return { saved: false, errors: {}, message: outputs }
    }
    const settings = { ...read.settings, version: read.settings.version || 1, outputs, ignore: ignoreFromValues(values) }
    await deps.engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot: deps.workspaceRoot, settings })
  } catch (error) {
    return { saved: false, errors: {}, message: error instanceof Error ? error.message : String(error) }
  }

  return { saved: true }
}

/** The outputs after the change, or a message when the edited storage has gone. */
function placeOutput (read: SettingsReadResult, index: number | undefined, values: StorageFormValues): Output[] | string {
  const current = read.settings.outputs
  if (index === undefined) {
    const created = valuesToOutput(values)

    return read.firstSync ? [created] : [...current, created]
  }
  if (index < 0 || index >= current.length) {
    return 'That storage is no longer in your LoreMaster settings.'
  }

  return current.map((output, position) => position === index ? valuesToOutput(values, output) : output)
}
