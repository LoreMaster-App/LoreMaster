import type { Output, Settings } from '../engine-protocol'
import { emptyValues, ignoreFromValues, outputToValues, valuesToOutput } from './storage-form-values.mapper'

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
