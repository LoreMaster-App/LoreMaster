import type { EngineClient } from '../engine-process'
import { type Generator, type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type Settings } from '../engine-protocol'
import { saveGenerator } from './save-generator.use-case'

const storage = (roots: string[]): Output => ({
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots, template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
})

function engine (settings: Settings, refuse?: string): { requests: EngineClient, saves: Settings[] } {
  const saves: Settings[] = []
  const requests = {
    request (method: string, params?: unknown) {
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings } as never)
      }
      if (method === SETTINGS_SAVE_METHOD) {
        if (refuse) {
          return Promise.reject(new Error(refuse))
        }
        saves.push((params as { settings: Settings }).settings)
      }

      return Promise.resolve(null as never)
    },
  } as unknown as EngineClient

  return { requests, saves }
}

const settings = (generators: Generator[], roots = ['docs']): Settings => ({ version: 1, outputs: [storage(roots)], generators })
const form = { kind: 'go-docs', output: 'docs/api', reads: 'libs/, apps/', command: '', title: '' }

describe('saveGenerator', () => {
  it('adds a generator and says where it went', async () => {
    const { requests, saves } = engine(settings([]))

    const result = await saveGenerator({ engine: requests, workspaceRoot: '/w' }, form)

    expect(result).toMatchObject({ saved: true, index: 0, generator: { type: 'go-docs', output: 'docs/api', input: ['libs/', 'apps/'] } })
    expect(saves[0].generators).toEqual([{ type: 'go-docs', output: 'docs/api', input: ['libs/', 'apps/'] }])
  })

  it('warns when the folder is outside every folder the storages sync', async () => {
    const { requests } = engine(settings([], ['guides']))

    const result = await saveGenerator({ engine: requests, workspaceRoot: '/w' }, form)

    expect(result).toMatchObject({ saved: true, notSynced: expect.stringContaining('is not inside a folder any storage syncs') })
  })

  it('refuses invalid values without writing', async () => {
    const { requests, saves } = engine(settings([{ type: 'ts-docs', output: 'docs/api' }]))

    const result = await saveGenerator({ engine: requests, workspaceRoot: '/w' }, form)

    expect(result).toMatchObject({ saved: false, errors: { output: expect.stringContaining('already writes') } })
    expect(saves).toHaveLength(0)
  })

  it('changes a generator in place, clearing what was emptied, and keeps its own folder allowed', async () => {
    const { requests, saves } = engine(settings([{ type: 'go-docs', output: 'docs/api', input: ['libs/'], title: 'Old', prepare: 'make' }]))

    const result = await saveGenerator({ engine: requests, workspaceRoot: '/w', index: 0 }, { ...form, reads: '', title: '', command: '' })

    expect(result).toMatchObject({ saved: true, index: 0 })
    expect(saves[0].generators).toEqual([{ type: 'go-docs', output: 'docs/api' }])
  })

  it('reports an engine refusal as the message', async () => {
    const { requests } = engine(settings([]), 'output must be inside the workspace')

    expect(await saveGenerator({ engine: requests, workspaceRoot: '/w' }, form)).toEqual({ saved: false, errors: {}, message: 'output must be inside the workspace' })
  })

  it('says so when the generator was removed meanwhile', async () => {
    const { requests } = engine(settings([]))

    expect(await saveGenerator({ engine: requests, workspaceRoot: '/w', index: 3 }, form)).toMatchObject({ saved: false, message: expect.stringContaining('no longer') })
  })
})
