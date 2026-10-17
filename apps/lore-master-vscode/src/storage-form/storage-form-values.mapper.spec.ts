import { describedConfig } from '../../test/config-description/config-description.fixture'
import type { Output, Settings } from '../engine-protocol'
import { emptyValues, formDescription, ignoreFromValues, outputToValues, valuesToOutput } from './storage-form-values.mapper'

const description = describedConfig()

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

describe('emptyValues', () => {
  it('starts with no platform and every choice on the default the description gives', () => {
    expect(emptyValues(description)).toMatchObject({ platform: '', direction: 'to-platform', mermaidMode: 'image', linkMode: 'title', titleCollision: 'fail', roots: '.', excludes: '', space: '', branch: '' })
  })
})

describe('formDescription', () => {
  it('hands the form the platforms, the shared fields and the ignore list setting', () => {
    const shown = formDescription(description)

    expect(shown.platforms.map(platform => platform.id)).toEqual(['confluence', 'github-pages', 'github-wiki'])
    expect(shown.contentFields.map(field => field.key)).toEqual(['roots', 'excludes'])
    expect(shown.ignoreField?.key).toBe('ignore')
  })
})

describe('outputToValues', () => {
  it('shows every field of a Confluence storage and the workspace ignore list', () => {
    expect(outputToValues(description, confluence, settings)).toMatchObject({
      name:           '',
      platform:       'confluence',
      baseUrl:        'https://acme.atlassian.net/wiki',
      space:          'ENG',
      parentPageId:   '123',
      titlePrefix:    'ENG',
      direction:      'two-way',
      mermaidMode:    'code',
      linkMode:       'id',
      titleCollision: 'adopt',
      template:       '',
      roots:          'docs, guides',
      excludes:       'drafts/',
      ignore:         'internal/',
    })
  })

  it('shows the default for a choice the storage leaves empty', () => {
    const values = outputToValues(description, { ...confluence, direction: '', mermaidMode: '', linkMode: '', titleCollision: '' }, { version: 1, outputs: [] })

    expect(values).toMatchObject({ direction: 'to-platform', mermaidMode: 'image', linkMode: 'title', titleCollision: 'fail', ignore: '' })
  })

  it('shows the list settings of an output as comma separated text', () => {
    expect(outputToValues(description, { ...confluence, include: ['keep.md', 'docs/'], exclude: ['tmp/'] }, settings)).toMatchObject({ include: 'keep.md, docs/', exclude: 'tmp/' })
  })
})

describe('valuesToOutput', () => {
  it('changes the space and parent of an existing storage and keeps what the form does not show', () => {
    const base: Output = { ...confluence, content: [{ type: 'markdown', roots: ['docs'], template: 'wiki' }, { type: 'openapi', roots: ['api'], template: 'default' }] }
    const values = { ...outputToValues(description, base, settings), space: 'OPS', parentPageId: '456', roots: 'docs, more', excludes: 'tmp/' }

    const output = valuesToOutput(description, values, base)

    expect(output).toMatchObject({ space: 'OPS', parentPageId: '456', titlePrefix: 'ENG' })
    expect(output.content).toEqual([
      { type: 'markdown', roots: ['docs', 'more'], excludes: ['tmp/'], template: 'wiki' },
      { type: 'openapi', roots: ['api'], template: 'default' },
    ])
  })

  it('builds a new GitHub Pages storage with nothing of Confluence in it', () => {
    const output = valuesToOutput(description, { ...emptyValues(description), platform: 'github-pages', repo: ' acme/docs ', branch: 'site', path: 'docs' })

    expect(output).toMatchObject({ platform: 'github-pages', baseUrl: '', space: '', parentPageId: '', repo: 'acme/docs', branch: 'site', path: 'docs' })
    expect(output.content[0]).toMatchObject({ roots: ['.'] })
  })

  it('clears what belongs to the other platforms when a storage changes platform', () => {
    const output = valuesToOutput(description, { ...outputToValues(description, confluence, settings), platform: 'github-wiki', repo: 'acme/docs' }, confluence)

    expect(output).toMatchObject({ platform: 'github-wiki', repo: 'acme/docs', baseUrl: '', space: '', parentPageId: '', titlePrefix: '' })
  })

  it('leaves the folder path out of a wiki storage', () => {
    const output = valuesToOutput(description, { ...emptyValues(description), platform: 'github-wiki', repo: 'acme/docs', path: 'ignored' })

    expect(output.path).toBe('')
  })

  it('falls back to the whole workspace when no folder is given', () => {
    const output = valuesToOutput(description, { ...emptyValues(description), platform: 'github-pages', roots: ' ' })

    expect(output.content[0].roots).toEqual(['.'])
  })

  it('stores a list setting as a list, and drops it when emptied', () => {
    const base: Output = { ...confluence, include: ['old.md'] }
    const values = outputToValues(description, base, settings)

    expect(valuesToOutput(description, { ...values, include: 'a.md, b/' }, base).include).toEqual(['a.md', 'b/'])
    expect(valuesToOutput(description, { ...values, include: '' }, base).include).toBeUndefined()
  })

  it('writes the settings of a platform the extension was never taught, from its description', () => {
    const added = describedConfig()
    added.platforms.push({
      id:              'notion',
      label:           'Notion',
      description:     '',
      icon:            'book',
      delivery:        'pages',
      nameTemplate:    '',
      summaryTemplate: '',
      fields:          [{ key: 'workspace', label: 'Workspace', type: 'text' }],
    })

    const output = valuesToOutput(added, { ...emptyValues(added), platform: 'notion', workspace: ' acme ' })

    expect(output).toMatchObject({ platform: 'notion', workspace: 'acme' })
  })
})

describe('the page template', () => {
  it('is shown as written and saved on the storage', () => {
    const base: Output = { ...confluence, template: '12345' }
    const values = outputToValues(description, base, settings)

    expect(values.template).toBe('12345')
    expect(valuesToOutput(description, { ...values, template: ' templates/layout.xhtml ' }, base).template).toBe('templates/layout.xhtml')
  })
})

describe('the storage name', () => {
  it('is shown and saved, and an empty one is not written', () => {
    const named: Output = { ...confluence, name: 'Technical documentation' }
    const values = outputToValues(description, named, settings)

    expect(values.name).toBe('Technical documentation')
    expect(valuesToOutput(description, { ...values, name: ' User manuals ' }, named).name).toBe('User manuals')
    expect(valuesToOutput(description, { ...values, name: '' }, named).name).toBeUndefined()
  })
})

describe('ignoreFromValues', () => {
  it('returns the patterns, or nothing when the field is empty', () => {
    expect(ignoreFromValues({ ...emptyValues(description), ignore: 'a/, b.md' })).toEqual(['a/', 'b.md'])
    expect(ignoreFromValues(emptyValues(description))).toBeUndefined()
  })
})
