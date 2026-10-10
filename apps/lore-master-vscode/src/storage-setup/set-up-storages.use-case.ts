import type { EngineClient } from '../engine-process'
import type { Output } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import type { SyncTargetUI } from '../sync-target'
import { configureConfluence } from './configure-confluence.use-case'
import { configureGitHubPages } from './configure-github-pages.use-case'
import { configureGitHubWiki } from './configure-github-wiki.use-case'

/** A storage type the first-run setup can configure. */
export type StorageType = 'confluence' | 'github-pages' | 'github-wiki'

/** What configuring a Confluence output asks of the editor: the target pickers, plus
 *  choosing or being told there is no connection. */
export interface ConfluenceSetupUI extends SyncTargetUI {
  pickConnection (connections: ConnectionMeta[]): Promise<ConnectionMeta | undefined>
  noConnections (): Promise<void>
  error (message: string): Promise<void>
}

/** What configuring a GitHub Pages output asks of the editor. */
export interface GitHubPagesSetupUI {
  promptRepo (): Promise<string | undefined>
  promptBranch (): Promise<string | undefined>
}

/** Everything the first-run setup asks of the editor. */
export interface StorageSetupUI extends ConfluenceSetupUI, GitHubPagesSetupUI {
  pickStorageTypes (): Promise<StorageType[] | undefined>
}

/** What the first-run setup needs. `addConnection`, when given, lets Confluence setup start
 *  the Add Connection wizard inline when there is no connection yet. */
export interface StorageSetupDeps {
  engine:         EngineClient
  connections:    ConnectionStore
  ui:             StorageSetupUI
  addConnection?: () => Promise<ConnectionMeta | undefined>
}

/**
 * The first-run "where to sync": pick one or more storage types, configure each in turn, and
 * return the outputs to save. Returns undefined when the user cancels or configures nothing,
 * so the caller writes nothing.
 */
export async function setUpStorages (deps: StorageSetupDeps): Promise<Output[] | undefined> {
  const { engine, connections, ui, addConnection } = deps

  const types = await ui.pickStorageTypes()
  if (!types || types.length === 0) {
    return undefined
  }

  const outputs: Output[] = []
  if (types.includes('confluence')) {
    const output = await configureConfluence({ engine, connections, ui, addConnection })
    if (output) {
      outputs.push(output)
    }
  }
  if (types.includes('github-pages')) {
    const output = await configureGitHubPages(ui)
    if (output) {
      outputs.push(output)
    }
  }

  if (types.includes('github-wiki')) {
    const output = await configureGitHubWiki(ui)
    if (output) {
      outputs.push(output)
    }
  }

  return outputs.length > 0 ? outputs : undefined
}
