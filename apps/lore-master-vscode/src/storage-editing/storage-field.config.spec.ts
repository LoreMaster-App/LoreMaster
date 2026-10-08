import type { Output } from '../engine-protocol'
import { fieldsFor, parseList, type StorageField } from './storage-field.config'

const confluence: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['docs', 'guides'], excludes: ['drafts/'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}

const pages: Output = { ...confluence, platform: 'github-pages', baseUrl: '', space: '', repo: 'acme/site', branch: 'live' }

const field = (output: Output, key: string): StorageField => {
  const found = fieldsFor(output).find(each => each.key === key)
  if (!found) {
    throw new Error(`no field ${key}`)
  }

  return found
}

describe('parseList', () => {
  it('splits on commas and new lines, trims, and drops empty entries', () => {
    expect(parseList(' docs , guides,\n\n notes/ ,')).toEqual(['docs', 'guides', 'notes/'])
    expect(parseList('')).toEqual([])
  })
})

describe('fieldsFor', () => {
  it('offers the Confluence settings for a Confluence storage', () => {
    expect(fieldsFor(confluence).map(each => each.key)).toEqual(['titlePrefix', 'direction', 'mermaidMode', 'linkMode', 'titleCollision', 'roots', 'excludes'])
  })

  it('offers the repository and branch for GitHub Pages, and no Confluence settings', () => {
    expect(fieldsFor(pages).map(each => each.key)).toEqual(['repo', 'branch', 'roots', 'excludes'])
  })

  it('reads the current values the way a user would type them', () => {
    expect(field(confluence, 'titlePrefix').read(confluence)).toBe('ENG')
    expect(field(confluence, 'direction').read(confluence)).toBe('to-platform')
    expect(field(confluence, 'roots').read(confluence)).toBe('docs, guides')
    expect(field(confluence, 'excludes').read(confluence)).toBe('drafts/')
    expect(field(pages, 'repo').read(pages)).toBe('acme/site')
    expect(field({ ...pages, repo: undefined }, 'repo').read({ ...pages, repo: undefined })).toBe('')
  })

  it('lists the allowed values of each enum', () => {
    expect(field(confluence, 'direction').options).toEqual(['to-platform', 'two-way'])
    expect(field(confluence, 'mermaidMode').options).toEqual(['image', 'code'])
    expect(field(confluence, 'linkMode').options).toEqual(['title', 'id'])
    expect(field(confluence, 'titleCollision').options).toEqual(['fail', 'adopt'])
  })

  it('writes a value without touching anything else, and without changing the original', () => {
    const changed = field(confluence, 'direction').write(confluence, 'two-way')

    expect(changed).toEqual({ ...confluence, direction: 'two-way' })
    expect(confluence.direction).toBe('to-platform')
  })

  it('trims text and writes the GitHub Pages fields', () => {
    expect(field(confluence, 'titlePrefix').write(confluence, '  OPS ').titlePrefix).toBe('OPS')
    expect(field(pages, 'branch').write(pages, ' docs-site ').branch).toBe('docs-site')
    expect(field(pages, 'repo').write(pages, '').repo).toBe('')
  })

  it('writes the folders and excludes into the Markdown content entry', () => {
    const rooted = field(confluence, 'roots').write(confluence, 'handbook, adr')
    expect(rooted.content[0].roots).toEqual(['handbook', 'adr'])
    expect(rooted.content[0].excludes).toEqual(['drafts/'])

    const excluded = field(confluence, 'excludes').write(confluence, 'internal/, *.tmp.md')
    expect(excluded.content[0].excludes).toEqual(['internal/', '*.tmp.md'])
    expect(excluded.content[0].roots).toEqual(['docs', 'guides'])
  })

  it('means the whole workspace when the folders are cleared', () => {
    expect(field(confluence, 'roots').write(confluence, '  ').content[0].roots).toEqual(['.'])
  })

  it('leaves content entries of other types alone', () => {
    const mixed: Output = { ...confluence, content: [{ type: 'test-results', roots: ['x'], template: 'default' }, ...confluence.content] }
    const changed = field(mixed, 'roots').write(mixed, 'only')

    expect(changed.content[0].roots).toEqual(['x'])
    expect(changed.content[1].roots).toEqual(['only'])
  })
})
