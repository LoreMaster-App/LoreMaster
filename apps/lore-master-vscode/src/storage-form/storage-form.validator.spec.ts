import { describedConfig } from '../../test/config-description/config-description.fixture'
import { emptyValues } from './storage-form-values.mapper'
import { validateStorageForm } from './storage-form.validator'

const description = describedConfig()
const confluence = { ...emptyValues(description), platform: 'confluence', baseUrl: 'https://x.atlassian.net/wiki', space: 'ENG', parentPageId: '123', titlePrefix: 'ENG' }

describe('validateStorageForm', () => {
  it('asks for a platform first', () => {
    expect(validateStorageForm(description, emptyValues(description), 1)).toEqual({ platform: 'Choose where to sync.' })
  })

  it('accepts a complete Confluence storage', () => {
    expect(validateStorageForm(description, confluence, 1)).toEqual({})
  })

  it('names the Confluence fields that are missing or wrong, from the description', () => {
    const errors = validateStorageForm(description, { ...confluence, baseUrl: '', space: ' ', parentPageId: 'abc', titlePrefix: '' }, 0)

    expect(Object.keys(errors).sort((a, b) => a.localeCompare(b))).toEqual(['baseUrl', 'parentPageId', 'space', 'titlePrefix'])
    expect(errors.baseUrl).toContain('Add a connection')
    expect(errors.parentPageId).toContain('digits only')
  })

  it('asks to choose the site when there are connections to choose from', () => {
    expect(validateStorageForm(description, { ...confluence, baseUrl: '' }, 2).baseUrl).toBe('Choose the Confluence address.')
  })

  it('accepts a GitHub storage with no repository, and refuses a malformed one', () => {
    expect(validateStorageForm(description, { ...emptyValues(description), platform: 'github-pages' }, 0)).toEqual({})
    expect(validateStorageForm(description, { ...emptyValues(description), platform: 'github-wiki', repo: 'acme/docs' }, 0)).toEqual({})
    expect(validateStorageForm(description, { ...emptyValues(description), platform: 'github-pages', repo: 'not a repo' }, 0).repo).toContain('owner/name')
  })

  it('knows no platform of its own: one the description adds is checked the same way', () => {
    const added = describedConfig()
    added.platforms.push({
      id:              'notion',
      label:           'Notion',
      description:     '',
      icon:            'book',
      delivery:        'pages',
      nameTemplate:    '',
      summaryTemplate: '',
      fields:          [{ key: 'workspace', label: 'Workspace', type: 'text', required: true }],
    })

    expect(validateStorageForm(added, { ...emptyValues(added), platform: 'notion' }, 0)).toEqual({ workspace: 'Workspace is required.' })
  })
})
