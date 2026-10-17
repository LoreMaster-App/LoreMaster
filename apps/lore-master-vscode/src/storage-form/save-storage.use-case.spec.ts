import { describedConfig } from '../../test/config-description/config-description.fixture'
import { describeRoutes, readResultOf } from '../../test/config-description/engine-answers.fixture'
import { CONFIG_DESCRIBE_METHOD, type Output, type Settings, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD } from '../engine-protocol'
import { saveStorage, type SaveStorageEngine } from './save-storage.use-case'
import { emptyValues } from './storage-form-values.mapper'

const existing: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
}

function engine (settings: Settings, firstSync = false, refuse?: string) {
  const saves: Settings[] = []
  const requests: SaveStorageEngine = {
    request (method: string, params?: unknown) {
      if (method === CONFIG_DESCRIBE_METHOD) {
        return Promise.resolve(describeRoutes()[method]() as never)
      }
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve(readResultOf(settings, { firstSync }) as never)
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

  return { requests, saves }
}

const description = describedConfig()
const confluenceValues = { ...emptyValues(description), platform: 'confluence', baseUrl: 'https://x.atlassian.net/wiki', space: 'OPS', parentPageId: '99', titlePrefix: 'OPS' }

describe('saveStorage', () => {
  it('appends a new storage beside the existing ones', async () => {
    const { requests, saves } = engine({ version: 1, outputs: [existing] })

    const result = await saveStorage({ engine: requests, workspaceRoot: '/w', connectionCount: 1 }, confluenceValues)

    expect(result).toEqual({ saved: true })
    expect(saves[0].outputs).toHaveLength(2)
    expect(saves[0].outputs[1]).toMatchObject({ space: 'OPS', parentPageId: '99' })
  })

  it('replaces the unconfigured scaffold of a fresh workspace', async () => {
    const { requests, saves } = engine({ version: 1, outputs: [existing] }, true)

    await saveStorage({ engine: requests, workspaceRoot: '/w', connectionCount: 1 }, confluenceValues)

    expect(saves[0].outputs).toHaveLength(1)
  })

  it('changes the space and parent of the edited storage only, and sets the ignore list', async () => {
    const other = { ...existing, space: 'DOC' }
    const { requests, saves } = engine({ version: 1, outputs: [existing, other] })

    await saveStorage({ engine: requests, workspaceRoot: '/w', index: 0, connectionCount: 1 }, { ...confluenceValues, ignore: 'internal/' })

    expect(saves[0].outputs[0]).toMatchObject({ space: 'OPS', parentPageId: '99' })
    expect(saves[0].outputs[1]).toEqual(other)
    expect(saves[0].ignore).toEqual(['internal/'])
  })

  it('returns the field errors without touching the settings', async () => {
    const { requests, saves } = engine({ version: 1, outputs: [] })

    const result = await saveStorage({ engine: requests, workspaceRoot: '/w', connectionCount: 1 }, { ...confluenceValues, space: '' })

    expect(result).toMatchObject({ saved: false, errors: { space: expect.any(String) } })
    expect(saves).toEqual([])
  })

  it("returns the engine's reason when it refuses the settings", async () => {
    const { requests } = engine({ version: 1, outputs: [] }, false, 'outputs[0].space is unknown')

    const result = await saveStorage({ engine: requests, workspaceRoot: '/w', connectionCount: 1 }, confluenceValues)

    expect(result).toEqual({ saved: false, errors: {}, message: 'outputs[0].space is unknown' })
  })

  it('says so when the edited storage is gone', async () => {
    const { requests } = engine({ version: 1, outputs: [] })

    const result = await saveStorage({ engine: requests, workspaceRoot: '/w', index: 3, connectionCount: 1 }, confluenceValues)

    expect(result).toMatchObject({ saved: false, message: expect.stringContaining('no longer') })
  })
})

describe('saveStorage names', () => {
  it('leaves an unnamed storage unnamed: the engine makes a name up from where it goes', async () => {
    const { requests, saves } = engine({ version: 1, outputs: [existing] })

    await saveStorage({ engine: requests, workspaceRoot: '/w', connectionCount: 1 }, confluenceValues)

    expect(saves[0].outputs[1].name).toBeUndefined()
  })

  it('saves the name the user gave', async () => {
    const { requests, saves } = engine({ version: 1, outputs: [] })

    await saveStorage({ engine: requests, workspaceRoot: '/w', connectionCount: 1 }, { ...confluenceValues, name: ' Technical docs ' })

    expect(saves[0].outputs[0].name).toBe('Technical docs')
  })

  it("shows the engine's refusal of a name another storage already has", async () => {
    const { requests } = engine({ version: 1, outputs: [{ ...existing, name: 'Technical docs' }] }, false, 'outputs[1].name "technical DOCS" is already the name of outputs[0]; names must be different')

    const result = await saveStorage({ engine: requests, workspaceRoot: '/w', connectionCount: 1 }, { ...confluenceValues, name: 'technical DOCS' })

    expect(result).toMatchObject({ saved: false, message: expect.stringContaining('already the name') })
  })
})
