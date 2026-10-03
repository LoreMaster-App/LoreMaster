import * as vscode from 'vscode'
import type { Page, Space } from '../engine-protocol'
import type { ParentAction, ParentPagePickerUI } from './parent-page-picker.handler'
import type { SpacePickerUI } from './space-picker.handler'
import type { TitlePrefixUI } from './title-prefix-prompt.handler'

/** The VS Code-backed UI for the whole sync-target flow. */
export type SyncTargetUI = SpacePickerUI & ParentPagePickerUI & TitlePrefixUI

const USE_THIS_PAGE = '$(check) Use this page'
const GO_UP = '$(arrow-up) Up'
const SEARCH_BY_TITLE = '$(search) Search by title'

export function createSyncTargetUI (): SyncTargetUI {
  return {
    async pickSpace (spaces: Space[]) {
      const picked = await vscode.window.showQuickPick(
        spaces.map(space => ({ label: space.name, description: space.key, space })),
        { title: 'Lore Master: space', placeHolder: 'Which Confluence space?' },
      )

      return picked?.space
    },

    async pickAction (view): Promise<ParentAction> {
      const items: { label: string; page?: Page }[] = [
        { label: USE_THIS_PAGE },
        ...(view.canGoUp ? [{ label: GO_UP }] : []),
        { label: SEARCH_BY_TITLE },
        ...view.children.map(page => ({ label: `$(file) ${page.title}`, page })),
      ]
      const picked = await vscode.window.showQuickPick(items, {
        title:       `Parent page — ${view.breadcrumb}`,
        placeHolder: 'Choose a parent page, or browse into one',
      })

      if (!picked) {
        return { kind: 'cancel' }
      }
      if (picked.page) {
        return { kind: 'descend', page: picked.page }
      }
      if (picked.label === USE_THIS_PAGE) {
        return { kind: 'use' }
      }
      if (picked.label === GO_UP) {
        return { kind: 'up' }
      }

      return { kind: 'search' }
    },

    promptSearchQuery () {
      return Promise.resolve(vscode.window.showInputBox({ title: 'Search pages by title', prompt: 'Title contains', ignoreFocusOut: true }))
    },

    async pickSearchResult (results: Page[]) {
      const picked = await vscode.window.showQuickPick(
        results.map(page => ({ label: page.title, description: page.url, page })),
        { title: 'Search results', placeHolder: results.length === 0 ? 'No pages matched' : 'Pick a parent page' },
      )

      return picked?.page
    },

    promptTitlePrefix (defaultValue: string, validate: (value: string) => string | undefined) {
      return Promise.resolve(vscode.window.showInputBox({
        title:          'Lore Master: page title prefix',
        prompt:         'Every synced page is titled "<prefix>: <heading>"',
        value:          defaultValue,
        ignoreFocusOut: true,
        validateInput:  value => validate(value),
      }))
    },
  }
}
