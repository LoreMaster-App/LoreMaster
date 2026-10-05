import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD } from '../engine-protocol'
import { removeStorage } from './remove-storage.use-case'
import type { StoragesEngine } from './storages-view.client'

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

const pages: Output = {
  platform:       'github-pages',
  baseUrl:        '',
  space:          '',
  parentPageId:   '',
  titlePrefix:    '',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['docs'], template: 'default' }],
  mermaidMode:    '',
  titleCollision: '',
  linkMode:       '',
  repo:           'owner/name',
  branch:         'gh-pages',
}

function engine (outputs: Output[], onSave: (params: unknown) => void): StoragesEngine {
  return {
    request (method: string, params?: unknown) {
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs } } as never)
      }
      if (method === SETTINGS_SAVE_METHOD) {
        onSave(params)
      }

      return Promise.resolve(null as never)
    },
  }
}

describe('removeStorage', () => {
  it('removes the output at the index and saves the rest', async () => {
    let saved: unknown
    const result = await removeStorage({ engine: engine([confluence, pages], params => { saved = params }), workspaceRoot: '/w', index: 0 })

    expect(result).toBe('removed')
    expect((saved as { settings: { outputs: Output[] } }).settings.outputs).toEqual([pages])
  })

  it('refuses to remove the only output (config must keep at least one)', async () => {
    let saved = false
    const result = await removeStorage({ engine: engine([confluence], () => { saved = true }), workspaceRoot: '/w', index: 0 })

    expect(result).toBe('last')
    expect(saved).toBe(false)
  })
})
