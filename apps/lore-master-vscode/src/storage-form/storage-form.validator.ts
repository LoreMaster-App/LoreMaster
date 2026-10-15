import type { StorageFormValues } from './storage-form.contract'

export type StorageFormErrors = Partial<Record<keyof StorageFormValues, string>>

/** What is wrong with the form, by field; an empty object means it can be saved. */
export function validateStorageForm (values: StorageFormValues, connectionCount: number, otherNames: string[] = []): StorageFormErrors {
  const errors: StorageFormErrors = {}

  const name = values.name.trim()
  if (name.length > 60) {
    errors.name = 'Use a name of at most 60 characters.'
  } else if (name !== '' && otherNames.some(other => other.trim().toLowerCase() === name.toLowerCase())) {
    errors.name = `Another storage is already called “${name}”. Names tell the storages apart, so each needs its own.`
  }

  if (values.platform === '') {
    errors.platform = 'Choose where to sync.'

    return errors
  }
  if (values.platform === 'confluence') {
    if (values.connection.trim() === '') {
      errors.connection = connectionCount === 0 ? 'Add a connection first.' : 'Choose the Confluence site.'
    }
    if (values.space.trim() === '') {
      errors.space = 'Name the space (its key, such as ENG).'
    }
    if (!/^\d+$/.test(values.parentPageId.trim())) {
      errors.parentPageId = 'The parent page is a page id: digits only. Search for it by title to fill it in.'
    }
  }
  if (values.platform !== 'confluence' && values.repo.trim() !== '' && !/^(?:[\w.-]+\/[\w.-]+|\S+:\/\/\S+|git@\S+)$/.test(values.repo.trim())) {
    errors.repo = 'Use owner/name or a clone URL, or leave it empty for this workspace\'s own origin.'
  }

  return errors
}
