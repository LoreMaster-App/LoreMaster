import {
  isGitHubOutput,
  type Output,
  SETTINGS_READ_METHOD,
  SETTINGS_SAVE_METHOD,
  type SettingsReadResult,
} from '../engine-protocol'
import { publishPagesOutput } from '../pages-command'
import { setUpStorages } from '../storage-setup'
import { type SyncDeps, syncConfluenceOutput } from './sync-run.use-case'

/** How a sync run chooses its outputs. */
export interface SyncOutputsOptions {
  /** Ask the user to pick a subset of the configured outputs, rather than syncing all. */
  choose?:         boolean
  /** Sync only Confluence outputs (the current-file sync, which GitHub Pages cannot scope). */
  confluenceOnly?: boolean
}

/**
 * Syncs the workspace's configured outputs, fanning out over each. The first run — no
 * syncable output yet — asks which storage types to set up, configures each, and saves them.
 * Confluence outputs go through the plan/preview/execute flow; GitHub Pages outputs are
 * published. "Sync to…" narrows to a chosen subset; the current-file sync narrows to
 * Confluence.
 */
export async function syncOutputs (deps: SyncDeps, options: SyncOutputsOptions = {}): Promise<void> {
  const { engine, connections, workspaceRoot, ui } = deps

  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  let outputs = read.settings.outputs
  let indices = outputs.map((_, index) => index).filter(index => isSyncable(outputs[index]))

  if (indices.length === 0) {
    const created = await setUpStorages({ engine, connections, ui, addConnection: deps.addConnection })
    if (!created || created.length === 0) {
      return
    }
    try {
      await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { version: read.settings.version || 1, outputs: created } })
    } catch (error) {
      await ui.error(messageOf(error))

      return
    }
    outputs = created
    indices = created.map((_, index) => index)
  }

  if (options.confluenceOnly) {
    indices = indices.filter(index => outputs[index].platform === 'confluence')
    if (indices.length === 0) {
      await ui.error('No Confluence output is configured, so there is nothing to sync the current file to.')

      return
    }
  }

  if (options.choose) {
    const chosen = await ui.pickOutputs(indices.map(index => ({ index, output: outputs[index] })))
    if (!chosen || chosen.length === 0) {
      return
    }
    indices = chosen
  }

  for (const index of indices) {
    const output = outputs[index]
    if (isGitHubOutput(output)) {
      await publishPagesOutput({ engine, workspaceRoot, output: index, ui })
    } else {
      await syncConfluenceOutput(deps, { output, index })
    }
  }
}

/** An output can be synced when it is a GitHub Pages output, or a Confluence output whose
 *  first-sync answers are all filled in. */
function isSyncable (output: Output): boolean {
  if (isGitHubOutput(output)) {
    return true
  }

  return output.platform === 'confluence' &&
    output.baseUrl !== '' &&
    output.space !== '' &&
    output.parentPageId !== '' &&
    output.titlePrefix !== ''
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
