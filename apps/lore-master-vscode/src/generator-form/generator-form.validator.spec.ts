import { allKinds } from './generator-form-values.mapper'
import { validateGeneratorForm } from './generator-form.validator'

const kinds = allKinds()
const base = { kind: 'go-docs', output: 'docs/api', reads: '', command: '', title: '' }

describe('validateGeneratorForm', () => {
  it('accepts a complete generator and one that reads the default', () => {
    expect(validateGeneratorForm(base, kinds, [])).toEqual({})
  })

  it('asks for a kind', () => {
    expect(validateGeneratorForm({ ...base, kind: '' }, kinds, [])).toEqual({ kind: expect.stringContaining('Choose') })
  })

  it('refuses a folder outside the workspace, the root, and one another generator writes to', () => {
    expect(validateGeneratorForm({ ...base, output: '../x' }, kinds, []).output).toContain('inside the workspace')
    expect(validateGeneratorForm({ ...base, output: '.' }, kinds, []).output).toContain('Name a folder')
    expect(validateGeneratorForm({ ...base, output: 'docs/api/' }, kinds, ['docs/api']).output).toContain('already writes to docs/api')
  })

  it('needs something to read for a Storybook, and a command and files for the recognised kind', () => {
    expect(validateGeneratorForm({ ...base, kind: 'storybook-docs', output: 'docs/sb' }, kinds, []).reads).toContain('Storybook')
    const recognised = validateGeneratorForm({ ...base, kind: 'recognise', output: 'docs/generated' }, kinds, [])
    expect(recognised.reads).toContain('Name what to read')
    expect(recognised.command).toContain('Name the command')
    expect(validateGeneratorForm({ ...base, kind: 'recognise', output: 'docs/generated', reads: 'build/', command: 'make' }, kinds, [])).toEqual({})
  })
})
