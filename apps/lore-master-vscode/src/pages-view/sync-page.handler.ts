import { type SyncCommandDeps, syncFile } from '../sync-command'
import type { PagesNode, PagesViewProvider } from './pages-view.client'

/** The command id contributed in package.json. */
export const SYNC_PAGE_COMMAND = 'loreMaster.syncPage'

/** Syncs the page a Pages view row stands for (and the ancestors it needs). */
export async function syncPage (deps: SyncCommandDeps, provider: PagesViewProvider, node: PagesNode | undefined): Promise<void> {
  if (node?.kind !== 'page') {
    return
  }
  const { folder } = await provider.configuredOutputs()
  if (folder === '') {
    return
  }

  await syncFile(deps, folder, node.entry.node.path)
  provider.refresh()
}
