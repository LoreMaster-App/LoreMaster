import { createTargetStore, type SyncTarget } from './sync-target.store'

const target: SyncTarget = { space: 'ENG', parentPageId: 'home', parentTitle: 'Engineering', titlePrefix: 'ENG' }

describe('createTargetStore', () => {
  it('keeps a target per folder', () => {
    const store = createTargetStore()

    store.set('/w/a', target)

    expect(store.get('/w/a')).toEqual(target)
    expect(store.get('/w/b')).toBeUndefined()
  })

  it('replaces a folder\'s target', () => {
    const store = createTargetStore()
    store.set('/w/a', target)

    store.set('/w/a', { ...target, titlePrefix: 'DOCS' })

    expect(store.get('/w/a')?.titlePrefix).toBe('DOCS')
  })
})
