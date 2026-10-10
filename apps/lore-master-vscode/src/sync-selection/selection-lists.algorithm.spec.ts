import type { Output, Settings } from '../engine-protocol'
import { excludeEntry, includeEntry, selectionEntry } from './selection-lists.algorithm'

function output (extra: Partial<Output> = {}): Output {
  return {
    platform:       'github-pages',
    baseUrl:        '',
    space:          '',
    parentPageId:   '',
    titlePrefix:    '',
    direction:      'to-platform',
    content:        [],
    mermaidMode:    'image',
    titleCollision: 'fail',
    linkMode:       'title',
    ...extra,
  }
}

function settings (...outputs: Output[]): Settings {
  return { version: 1, outputs }
}

describe('selectionEntry', () => {
  it('ends a folder in a slash and leaves a file alone', () => {
    expect(selectionEntry('docs/adr', true)).toBe('docs/adr/')
    expect(selectionEntry('docs/adr/', true)).toBe('docs/adr/')
    expect(selectionEntry('docs/a.md', false)).toBe('docs/a.md')
  })
})

describe('excludeEntry', () => {
  it('adds to the top-level ignore list for all storages, once', () => {
    const once = excludeEntry(settings(output()), 'notes/', { kind: 'all' })
    const twice = excludeEntry(once, 'notes/', { kind: 'all' })

    expect(twice.ignore).toEqual(['notes/'])
  })

  it('adds to the chosen storages only, and takes the entry out of their include', () => {
    const start = settings(output({ include: ['a.md'] }), output())
    const next = excludeEntry(start, 'a.md', { kind: 'storages', indexes: [0] })

    expect(next.outputs[0].exclude).toEqual(['a.md'])
    expect(next.outputs[0].include).toBeUndefined()
    expect(next.outputs[1].exclude).toBeUndefined()
  })
})

describe('includeEntry', () => {
  it('removes an exact exclusion of the storage instead of adding an include', () => {
    const next = includeEntry(settings(output({ exclude: ['a.md', 'b.md'] })), 'a.md', [0])

    expect(next.outputs[0].exclude).toEqual(['b.md'])
    expect(next.outputs[0].include).toBeUndefined()
  })

  it('adds an include when something broader leaves the file out', () => {
    const start: Settings = { ...settings(output({ exclude: ['internal/'] }), output()), ignore: ['notes/'] }
    const next = includeEntry(start, 'internal/faq.md', [0])

    expect(next.outputs[0].include).toEqual(['internal/faq.md'])
    expect(next.outputs[0].exclude).toEqual(['internal/'])
    expect(next.outputs[1].include).toBeUndefined()
    expect(next.ignore).toEqual(['notes/'])
  })
})
