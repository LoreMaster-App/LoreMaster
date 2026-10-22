import { fieldsOf } from '../config-description'
import type { ConfigDescription } from '../engine-protocol'
import type { StorageFormValues } from './storage-form.contract'

/** What is wrong with the form, by field key. */
export type StorageFormErrors = Record<string, string>

/**
 * What is wrong with the form, by field; an empty object means it can be saved. Only what the
 * engine's description states is checked here (a required field, a pattern); the engine validates
 * the rest when the storage is saved, and its refusal is shown to the user.
 */
/** Whether the text is a whole number inside the field's bounds. */
function inRange (value: string, field: { min?: number, max?: number }): boolean {
  const number = Number(value)

  return Number.isSafeInteger(number) && (field.min === undefined || number >= field.min) && (field.max === undefined || number <= field.max)
}

export function validateStorageForm (description: ConfigDescription, values: StorageFormValues, connectionCount: number): StorageFormErrors {
  const errors: StorageFormErrors = {}
  const platform = description.platforms.find(each => each.id === values.platform)
  if (!platform) {
    errors.platform = 'Choose where to sync.'

    return errors
  }

  for (const field of fieldsOf(description, platform)) {
    const value = (values[field.key] ?? '').trim()
    if (value === '') {
      if (field.required === true) {
        errors[field.key] = field.key === platform.connection?.field
          ? (connectionCount === 0 ? 'Add a connection first.' : `Choose the ${field.label}.`)
          : `${field.label} is required.`
      }
    } else if (field.pattern !== undefined && !new RegExp(field.pattern).test(value)) {
      errors[field.key] = field.patternHelp ?? `${field.label} is not valid.`
    } else if (field.type === 'number' && !inRange(value, field)) {
      errors[field.key] = `${field.label} must be a whole number${field.min === undefined ? '' : ` from ${field.min}`}${field.max === undefined ? '' : ` to ${field.max}`}.`
    }
  }

  return errors
}
