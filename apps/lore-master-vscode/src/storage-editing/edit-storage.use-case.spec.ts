import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD, type Settings } from '../engine-protocol'
import type { StoragesEngine } from '../sidebar'
import { editStorage } from './edit-storage.use-case'
import { fieldsFor } from './storage-field.config'

const confluence: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}
const second: Output = { ...confluence, space: 'OPS', titlePrefix: 'OPS' }

function engine (settings: Settings, saves: unknown[], refuse?: string): StoragesEngine {
  return {
    request (method: string, params?: unknown) {
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings } as never)
      }
      if (method === SETTINGS_SAVE_METHOD) {
        if (refuse) {
          return Promise.reject(new Error(refuse))
        }
        saves.push(params)
      }

      return Promise.resolve(null as never)
    },
  }
}

const field = (key: string) => fieldsFor(confluence).find(each => each.key === key)!

describe('editStorage', () => {
  it('changes one setting of one storage and saves everything else as it was', async () => {
    const saves: unknown[] = []
    const settings: Settings = { version: 1, skipGitignored: false, ignore: ['drafts/'], outputs: [confluence, second] }

    await editStorage({ engine: engine(settings, saves), workspaceRoot: '/w', index: 1 }, field('direction'), 'two-way')

    expect(saves).toEqual([{
      workspaceRoot: '/w',
      settings:      { version: 1, skipGitignored: false, ignore: ['drafts/'], outputs: [confluence, { ...second, direction: 'two-way' }] },
    }])
  })

  it('refuses a storage that is no longer there, writing nothing', async () => {
    const saves: unknown[] = []

    await expect(editStorage({ engine: engine({ version: 1, outputs: [confluence] }, saves), workspaceRoot: '/w', index: 3 }, field('direction'), 'two-way'))
      .rejects.toThrow('no longer in .lore-master.yaml')
    expect(saves).toEqual([])
  })

  it('passes the engine\'s reason on when it refuses the value', async () => {
    await expect(editStorage({ engine: engine({ version: 1, outputs: [confluence] }, [], 'outputs[0].mermaidMode "crayon" is not one of image, code'), workspaceRoot: '/w', index: 0 }, field('mermaidMode'), 'crayon'))
      .rejects.toThrow('crayon')
  })
})
