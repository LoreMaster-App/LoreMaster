import type { EngineRequester } from '../connection-setup'
import { PAGE_CHILDREN_METHOD, PAGE_SEARCH_METHOD, type Page, type Space } from '../engine-protocol'
import { type ParentAction, type ParentPagePickerUI, pickParentPage } from './parent-page-picker.handler'

const space: Space = { id: '1', key: 'ENG', name: 'Engineering', homepageId: 'home' }

function engineOf (childrenByPage: Record<string, Page[]>, searchResults: Page[] = []): EngineRequester {
  return {
    request: (method, params) => {
      if (method === PAGE_CHILDREN_METHOD) {
        return Promise.resolve({ pages: childrenByPage[(params as { pageId: string }).pageId] ?? [] } as never)
      }
      if (method === PAGE_SEARCH_METHOD) {
        return Promise.resolve({ pages: searchResults } as never)
      }

      throw new Error(`unexpected ${method}`)
    },
  }
}

function scriptedUI (actions: ParentAction[], search?: { query?: string; result?: Page }): ParentPagePickerUI & { breadcrumbs: string[] } {
  const breadcrumbs: string[] = []
  let next = 0

  return {
    breadcrumbs,
    pickAction: view => {
      breadcrumbs.push(view.breadcrumb)

      return Promise.resolve(actions[next++])
    },
    promptSearchQuery: () => Promise.resolve(search?.query),
    pickSearchResult:  () => Promise.resolve(search?.result),
  }
}

const pageA: Page = { id: 'a', title: 'Area A' }

describe('pickParentPage', () => {
  it('descends into a child then uses it, tracking the breadcrumb', async () => {
    const engine = engineOf({ home: [pageA], a: [] })
    const ui = scriptedUI([{ kind: 'descend', page: pageA }, { kind: 'use' }])

    const chosen = await pickParentPage({ engine, sessionId: 's', space, ui })

    expect(chosen).toEqual({ pageId: 'a', title: 'Area A' })
    expect(ui.breadcrumbs).toEqual(['Engineering', 'Engineering / Area A'])
  })

  it('goes back up and uses the homepage', async () => {
    const engine = engineOf({ home: [pageA], a: [] })
    const ui = scriptedUI([{ kind: 'descend', page: pageA }, { kind: 'up' }, { kind: 'use' }])

    const chosen = await pickParentPage({ engine, sessionId: 's', space, ui })

    expect(chosen).toEqual({ pageId: 'home', title: 'Engineering' })
  })

  it('jumps straight to a search result', async () => {
    const found: Page = { id: 'z', title: 'Found' }
    const engine = engineOf({ home: [] }, [found])
    const ui = scriptedUI([{ kind: 'search' }], { query: 'fou', result: found })

    const chosen = await pickParentPage({ engine, sessionId: 's', space, ui })

    expect(chosen).toEqual({ pageId: 'z', title: 'Found' })
  })

  it('opens in search when the space has no homepage', async () => {
    const found: Page = { id: 'z', title: 'Found' }
    const engine = engineOf({}, [found])
    const ui = scriptedUI([], { query: '', result: found })

    const chosen = await pickParentPage({ engine, sessionId: 's', space: { ...space, homepageId: undefined }, ui })

    expect(chosen).toEqual({ pageId: 'z', title: 'Found' })
  })

  it('returns undefined when cancelled', async () => {
    const engine = engineOf({ home: [] })
    const ui = scriptedUI([{ kind: 'cancel' }])

    expect(await pickParentPage({ engine, sessionId: 's', space, ui })).toBeUndefined()
  })
})
