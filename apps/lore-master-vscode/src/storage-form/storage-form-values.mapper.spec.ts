import type { Output, Settings } from '../engine-protocol'
import { emptyValues, ignoreFromValues, outputToValues, suggestStorageName, valuesToOutput } from './storage-form-values.mapper'

const confluence: Output = {
  platform:       'confluence',
  baseUrl:        'https://acme.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '123',
  titlePrefix:    'ENG',
  direction:      'two-way',
  mermaidMode:    'code',
  titleCollision: 'adopt',
  linkMode:       'id',
  content:        [{ type: 'markdown', roots: ['docs', 'guides'], excludes: ['drafts/'], template: 'default' }],
}

const settings: Settings = { version: 1, outputs: [confluence], ignore: ['internal/'] }

describe('outputToValues', () => {
  it('shows every field of a Confluence storage and the workspace ignore list', () => {
    expect(outputToValues(confluence, settings)).toEqual({
      name:           '',
      platform:       'confluence',
      connection:     'https://acme.atlassian.net/wiki',
      space:          'ENG',
      parentPageId:   '123',
      titlePrefix:    'ENG',
      direction:      'two-way',
      mermaidMode:    'code',
      linkMode:       'id',
      titleCollision: 'adopt',
      repo:           '',
      branch:         '',
      path:           '',
      template:       '',
      roots:          'docs, guides',
      excludes:       'drafts/',
      ignore:         'internal/',
    })
  })

  it('shows the built-in defaults for what the storage leaves empty', () => {
    const values = outputToValues({ ...confluence, direction: '', mermaidMode: '', linkMode: '', titleCollision: '' }, { version: 1, outputs: [] })

    expect(values).toMatchObject({ direction: 'to-platform', mermaidMode: 'image', linkMode: 'title', titleCollision: 'fail', ignore: '' })
  })
})

describe('valuesToOutput', () => {
  it('changes the space and parent of an existing storage and keeps what the form does not show', () => {
    const base: Output = { ...confluence, content: [{ type: 'markdown', roots: ['docs'], template: 'wiki' }, { type: 'openapi', roots: ['api'], template: 'default' }], include: ['keep.md'] }
    const values = { ...outputToValues(base, settings), space: 'OPS', parentPageId: '456', roots: 'docs, more', excludes: 'tmp/' }

    const output = valuesToOutput(values, base)

    expect(output).toMatchObject({ space: 'OPS', parentPageId: '456', titlePrefix: 'ENG', include: ['keep.md'] })
    expect(output.content).toEqual([
      { type: 'markdown', roots: ['docs', 'more'], excludes: ['tmp/'], template: 'wiki' },
      { type: 'openapi', roots: ['api'], template: 'default' },
    ])
  })

  it('builds a new GitHub Pages storage with no Confluence fields', () => {
    const output = valuesToOutput({ ...emptyValues(), platform: 'github-pages', repo: ' acme/docs ', branch: 'site', path: 'docs' })

    expect(output).toMatchObject({ platform: 'github-pages', baseUrl: '', space: '', parentPageId: '', repo: 'acme/docs', branch: 'site', path: 'docs' })
    expect(output.content[0]).toMatchObject({ roots: ['.'] })
  })

  it('leaves the folder path out of a wiki storage', () => {
    const output = valuesToOutput({ ...emptyValues(), platform: 'github-wiki', repo: 'acme/docs', path: 'ignored' })

    expect(output.path).toBeUndefined()
  })

  it('falls back to the whole workspace when no folder is given', () => {
    const output = valuesToOutput({ ...emptyValues(), platform: 'github-pages', roots: ' ' })

    expect(output.content[0].roots).toEqual(['.'])
  })
})

describe('ignoreFromValues', () => {
  it('returns the patterns, or nothing when the field is empty', () => {
    expect(ignoreFromValues({ ...emptyValues(), ignore: 'a/, b.md' })).toEqual(['a/', 'b.md'])
    expect(ignoreFromValues(emptyValues())).toBeUndefined()
  })
})

describe('the page template', () => {
  it('is shown as written and saved on the storage, and the default is not saved', () => {
    const base: Output = { ...confluence, template: '12345' }
    const values = outputToValues(base, settings)

    expect(values.template).toBe('12345')
    expect(valuesToOutput({ ...values, template: ' templates/layout.xhtml ' }, base).template).toBe('templates/layout.xhtml')
    expect(valuesToOutput({ ...values, template: 'default' }, base).template).toBeUndefined()
    expect(valuesToOutput({ ...values, template: '' }, base).template).toBeUndefined()
  })
})

describe('the storage name', () => {
  it('is shown and saved, and an empty one is not written', () => {
    const named: Output = { ...confluence, name: 'Technical documentation' }
    const values = outputToValues(named, settings)

    expect(values.name).toBe('Technical documentation')
    expect(valuesToOutput({ ...values, name: ' User manuals ' }, named).name).toBe('User manuals')
    expect(valuesToOutput({ ...values, name: '' }, named).name).toBeUndefined()
  })

  it('is suggested from where the storage goes, numbered when taken', () => {
    const values = { ...outputToValues(confluence, settings), name: '' }

    expect(suggestStorageName(values, [])).toBe('Confluence · ENG')
    expect(suggestStorageName(values, ['confluence · eng'])).toBe('Confluence · ENG (2)')
    expect(suggestStorageName(values, ['Confluence · ENG', 'Confluence · ENG (2)'])).toBe('Confluence · ENG (3)')
    expect(suggestStorageName({ ...values, platform: 'github-wiki' }, [])).toBe('GitHub Wiki · this repository')
    expect(suggestStorageName({ ...values, platform: 'github-pages', repo: 'acme/site' }, [])).toBe('GitHub Pages · acme/site')
  })
})
