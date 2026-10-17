import { type DisplayMemento, LocationDisplayStore } from './location-display.store'
import type { DisplayOptions } from './page-tree.contract'
import { rowContext } from './row-context.policy'

function memento (): DisplayMemento & { saved: Record<string, unknown> } {
  const saved: Record<string, unknown> = {}

  return {
    saved,
    get:    <T>(key: string) => saved[key] as T | undefined,
    update: (key, value) => {
      saved[key] = value

      return Promise.resolve()
    },
  }
}

const defaults: DisplayOptions = { label: 'title', view: 'storage', excluded: 'faded' }

describe('LocationDisplayStore', () => {
  it('shows a location by the defaults until it is given a choice of its own', () => {
    const store = new LocationDisplayStore(memento(), () => defaults)

    expect(store.optionsOf(0)).toEqual(defaults)
    expect(store.optionsOf(-1)).toEqual(defaults)
  })

  it('remembers a choice for one location only, and keeps its other choices on the defaults', async () => {
    const store = new LocationDisplayStore(memento(), () => defaults)

    await store.set(1, { view: 'flat' })
    await store.set(1, { label: 'fileName' })

    expect(store.optionsOf(1)).toEqual({ label: 'fileName', view: 'flat', excluded: 'faded' })
    expect(store.optionsOf(0)).toEqual(defaults)
  })

  it('follows the defaults for what a location never chose, even when they change', async () => {
    let current = defaults
    const store = new LocationDisplayStore(memento(), () => current)
    await store.set(0, { view: 'repo' })

    current = { ...defaults, excluded: 'hidden' }

    expect(store.optionsOf(0)).toEqual({ label: 'title', view: 'repo', excluded: 'hidden' })
  })

  it('forgets every choice on reset', async () => {
    const store = new LocationDisplayStore(memento(), () => defaults)
    await store.set(0, { view: 'flat' })

    await store.reset()

    expect(store.optionsOf(0)).toEqual(defaults)
  })
})

describe('rowContext', () => {
  it('names the kind of row and how it is shown, one word each, for the menus', () => {
    expect(rowContext('loreMasterStorage', { label: 'fileName', view: 'flat', excluded: 'hidden' })).toBe('loreMasterStorage label-fileName view-flat ignored-hidden')
    expect(rowContext('loreMasterLocal', defaults)).toBe('loreMasterLocal label-title view-storage ignored-faded')
  })
})
