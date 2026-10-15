import { emptyValues } from './storage-form-values.mapper'
import { validateStorageForm } from './storage-form.validator'

const confluence = { ...emptyValues(), platform: 'confluence' as const, connection: 'https://x.atlassian.net/wiki', space: 'ENG', parentPageId: '123' }

describe('validateStorageForm', () => {
  it('asks for a platform first', () => {
    expect(validateStorageForm(emptyValues(), 1)).toEqual({ platform: 'Choose where to sync.' })
  })

  it('accepts a complete Confluence storage', () => {
    expect(validateStorageForm(confluence, 1)).toEqual({})
  })

  it('names the Confluence fields that are missing or wrong', () => {
    const errors = validateStorageForm({ ...confluence, connection: '', space: ' ', parentPageId: 'abc' }, 0)

    expect(Object.keys(errors).sort((a, b) => a.localeCompare(b))).toEqual(['connection', 'parentPageId', 'space'])
    expect(errors.connection).toContain('Add a connection')
  })

  it('accepts a GitHub storage with no repository, and refuses a malformed one', () => {
    expect(validateStorageForm({ ...emptyValues(), platform: 'github-pages' }, 0)).toEqual({})
    expect(validateStorageForm({ ...emptyValues(), platform: 'github-wiki', repo: 'acme/docs' }, 0)).toEqual({})
    expect(validateStorageForm({ ...emptyValues(), platform: 'github-pages', repo: 'not a repo' }, 0).repo).toContain('owner/name')
  })
})
