import { allKinds, generatorToValues, kindOf, kindsFor, typeOfKind, valuesToGenerator } from './generator-form-values.mapper'

describe('the generator form values', () => {
  it('offers every kind, the typeless one included, with what each reads', () => {
    const kinds = allKinds()

    expect(kinds.map(kind => kind.kind)).toEqual(expect.arrayContaining(['recognise', 'test-results', 'storybook-docs', 'go-docs']))
    expect(kinds.find(kind => kind.kind === 'recognise')).toMatchObject({ readsRequired: true, command: 'required' })
    expect(kinds.find(kind => kind.kind === 'storybook-docs')).toMatchObject({ readsSource: true, readsRequired: true, defaultReads: 'storybook-static', command: 'optional' })
    expect(kinds.find(kind => kind.kind === 'go-docs')).toMatchObject({ readsSource: false, readsRequired: false, defaultOutput: 'docs/api' })
  })

  it('maps the typeless generator to and from `recognise`', () => {
    expect(kindOf('')).toBe('recognise')
    expect(typeOfKind('recognise')).toBe('')
    expect(typeOfKind('go-docs')).toBe('go-docs')
  })

  it('writes a generator without the fields left empty', () => {
    expect(valuesToGenerator({ kind: 'go-docs', output: String.raw` docs\api `, reads: ' libs/, apps/ ', command: '', title: '' }))
      .toEqual({ type: 'go-docs', output: 'docs/api', input: ['libs/', 'apps/'] })
    expect(valuesToGenerator({ kind: 'storybook-docs', output: 'docs/storybook', reads: 'storybook-static', command: 'npm run build-storybook', title: 'Components' }))
      .toEqual({ type: 'storybook-docs', output: 'docs/storybook', source: 'storybook-static', prepare: 'npm run build-storybook', title: 'Components' })
    expect(valuesToGenerator({ kind: 'recognise', output: 'docs/generated', reads: 'build/reports/', command: 'npm test', title: '' }))
      .toEqual({ type: '', output: 'docs/generated', input: ['build/reports/'], prepare: 'npm test' })
  })

  it('reads a generator back into the form, and offers only its own kind to edit', () => {
    const generator = { type: 'storybook-docs', output: 'docs/storybook', source: 'https://sb.example', prepare: 'npm run sb', title: 'UI' }

    expect(generatorToValues(generator)).toEqual({ kind: 'storybook-docs', output: 'docs/storybook', reads: 'https://sb.example', command: 'npm run sb', title: 'UI' })
    expect(generatorToValues({ type: 'go-docs', output: 'docs/api', input: ['a/', 'b/'] }).reads).toBe('a/, b/')
    expect(kindsFor(generator).map(kind => kind.kind)).toEqual(['storybook-docs'])
  })
})
