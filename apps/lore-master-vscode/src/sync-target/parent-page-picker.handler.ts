import { type Page, PAGE_CHILDREN_METHOD, PAGE_SEARCH_METHOD, type PagesResult, type Space } from '../engine-protocol'
import type { EngineRequester } from '../connection-setup'

/** A chosen parent page. */
export interface ParentPage {
  pageId: string
  title:  string
}

/** What the user does at one node of the tree walk. */
export type ParentAction =
  | { kind: 'descend'; page: Page } |
  { kind: 'use' } |
  { kind: 'up' } |
  { kind: 'search' } |
  { kind: 'cancel' }

/** The parent-page picker's prompts. The handler drives the tree walk; the UI only shows
 *  one level (or one search) at a time. */
export interface ParentPagePickerUI {
  pickAction (view: { breadcrumb: string; children: Page[]; canGoUp: boolean }): Promise<ParentAction>
  promptSearchQuery (): Promise<string | undefined>
  pickSearchResult (results: Page[]): Promise<Page | undefined>
}

/**
 * Walks the space's page tree from its homepage so the user can point at a parent page:
 * descend into a child, go up, "use this page", or search by title (jumping straight to a
 * result). Returns the chosen page, or undefined if cancelled. When the space has no
 * homepage, it opens in search.
 */
export async function pickParentPage (deps: { engine: EngineRequester; sessionId: string; space: Space; ui: ParentPagePickerUI }): Promise<ParentPage | undefined> {
  const { engine, sessionId, space, ui } = deps

  if (!space.homepageId) {
    return searchForParent(deps)
  }

  const parents: ParentPage[] = []
  let here: ParentPage = { pageId: space.homepageId, title: space.name }

  for (;;) {
    const { pages } = await engine.request<PagesResult>(PAGE_CHILDREN_METHOD, { sessionId, pageId: here.pageId })
    const action = await ui.pickAction({
      breadcrumb: [...parents, here].map(page => page.title).join(' / '),
      children:   pages,
      canGoUp:    parents.length > 0,
    })

    if (action.kind === 'use') {
      return here
    }
    if (action.kind === 'cancel') {
      return undefined
    }
    if (action.kind === 'descend') {
      parents.push(here)
      here = { pageId: action.page.id, title: action.page.title }
      continue
    }
    if (action.kind === 'up') {
      here = parents.pop() ?? here
      continue
    }

    const found = await searchForParent(deps)
    if (found) {
      return found
    }
  }
}

async function searchForParent (deps: { engine: EngineRequester; sessionId: string; space: Space; ui: ParentPagePickerUI }): Promise<ParentPage | undefined> {
  const query = await deps.ui.promptSearchQuery()
  if (query === undefined) {
    return undefined
  }
  const { pages } = await deps.engine.request<PagesResult>(PAGE_SEARCH_METHOD, { sessionId: deps.sessionId, spaceKey: deps.space.key, query })
  const chosen = await deps.ui.pickSearchResult(pages)

  return chosen && { pageId: chosen.id, title: chosen.title }
}
