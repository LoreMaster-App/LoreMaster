import { type Generator, type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type Settings } from '../engine-protocol'
import { addGenerator, type GeneratorsViewEngine, readGeneratorSettings, removeGenerator, updateGenerator } from './generators.use-case'

const output: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['docs'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}

const tests: Generator = { type: 'test-results', output: 'docs/tests', input: ['reports/'], title: 'CI results' }
const api: Generator = { type: 'go-docs', output: 'docs/api' }

function engine (settings: Settings, saves: Settings[] = [], refuse?: string): GeneratorsViewEngine {
  return {
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
  }
}

describe('readGeneratorSettings', () => {
  it('returns the generators and the storages, and no generators when there are none', async () => {
    expect(await readGeneratorSettings(engine({ version: 1, outputs: [output], generators: [tests] }), '/w')).toEqual({ generators: [tests], outputs: [output] })
    expect(await readGeneratorSettings(engine({ version: 1, outputs: [output] }), '/w')).toEqual({ generators: [], outputs: [output] })
  })
})

describe('addGenerator', () => {
  it('appends the generator, returns its index, and sends everything else as it was', async () => {
    const saves: Settings[] = []
    const settings: Settings = { version: 1, skipGitignored: false, ignore: ['drafts/'], outputs: [output], generators: [tests] }

    const index = await addGenerator(engine(settings, saves), '/w', api)

    expect(index).toBe(1)
    expect(saves).toEqual([{ ...settings, generators: [tests, api] }])
  })

  it('starts the list when there is none', async () => {
    const saves: Settings[] = []

    expect(await addGenerator(engine({ version: 1, outputs: [output] }, saves), '/w', api)).toBe(0)
    expect(saves[0].generators).toEqual([api])
  })

  it('passes the engine\'s reason on when it refuses the generator', async () => {
    await expect(addGenerator(engine({ version: 1, outputs: [output] }, [], 'generators[0].output "." must be a folder inside the workspace'), '/w', { ...api, output: '.' }))
      .rejects.toThrow('must be a folder inside the workspace')
  })
})

describe('updateGenerator', () => {
  const settings: Settings = { version: 1, outputs: [output], generators: [tests, api] }

  it('changes only what the patch names', async () => {
    const saves: Settings[] = []

    await updateGenerator(engine(settings, saves), '/w', 0, { output: 'docs/ci' })

    expect(saves[0].generators).toEqual([{ ...tests, output: 'docs/ci' }, api])
  })

  it('clears the input and the title by sending them absent', async () => {
    const saves: Settings[] = []

    await updateGenerator(engine(settings, saves), '/w', 0, { input: [], title: '' })

    expect(saves[0].generators?.[0]).toEqual({ type: 'test-results', output: 'docs/tests', input: undefined, title: undefined })
    expect(JSON.stringify(saves[0].generators?.[0])).toBe('{"type":"test-results","output":"docs/tests"}')
  })

  it('replaces the input and the title', async () => {
    const saves: Settings[] = []

    await updateGenerator(engine(settings, saves), '/w', 1, { input: ['libs/', '!**/internal/'], title: 'API' })

    expect(saves[0].generators?.[1]).toEqual({ type: 'go-docs', output: 'docs/api', input: ['libs/', '!**/internal/'], title: 'API' })
  })

  it('refuses a generator that is no longer there, writing nothing', async () => {
    const saves: Settings[] = []

    await expect(updateGenerator(engine(settings, saves), '/w', 5, { output: 'x' })).rejects.toThrow('no longer in .lore-master.yaml')
    expect(saves).toEqual([])
  })
})

describe('removeGenerator', () => {
  it('removes one generator and keeps the rest in order', async () => {
    const saves: Settings[] = []

    await removeGenerator(engine({ version: 1, outputs: [output], generators: [tests, api] }, saves), '/w', 0)

    expect(saves[0].generators).toEqual([api])
  })

  it('sends no list at all when the last generator goes, so the engine drops the key', async () => {
    const saves: Settings[] = []

    await removeGenerator(engine({ version: 1, outputs: [output], generators: [tests] }, saves), '/w', 0)

    expect(saves[0].generators).toBeUndefined()
    expect(JSON.stringify(saves[0])).not.toContain('generators')
  })

  it('refuses a generator that is no longer there', async () => {
    await expect(removeGenerator(engine({ version: 1, outputs: [output], generators: [tests] }), '/w', 3)).rejects.toThrow('no longer in .lore-master.yaml')
  })
})
