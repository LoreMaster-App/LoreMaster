import { describeConfig, isPagesOutput, isSiteOutput } from '../config-description'
import {
  type ConfigDescription,
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
  choose?:    boolean
  /** Sync only outputs that sync page by page (the current-file sync, which a published site cannot scope). */
  pagesOnly?: boolean
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
  const description = await describeConfig(engine)
  let outputs = read.settings.outputs
  let indices = outputs.map((_, index) => index).filter(index => isSyncable(description, outputs[index]))

  if (indices.length === 0) {
    const created = await setUpStorages({ engine, connections, ui, description, addConnection: deps.addConnection })
    if (!created || created.length === 0) {
      return
    }
    try {
      await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { ...read.settings, version: read.settings.version || 1, outputs: created } })
    } catch (error) {
      await ui.error(messageOf(error))

      return
    }
    outputs = created
    indices = created.map((_, index) => index)
  }

  if (options.pagesOnly) {
    indices = indices.filter(index => isPagesOutput(description, outputs[index]))
    if (indices.length === 0) {
      await ui.error('No output that syncs page by page is configured, so there is nothing to sync the current file to.')

      return
    }
  }

  if (options.choose) {
    const chosen = await ui.pickOutputs(indices.map(index => ({ index, output: outputs[index], label: read.displayNames?.[index] ?? outputs[index].platform, detail: read.summaries?.[index] ?? '' })))
    if (!chosen || chosen.length === 0) {
      return
    }
    indices = chosen
  }

  for (const index of indices) {
    const output = outputs[index]
    if (isSiteOutput(description, output)) {
      await publishPagesOutput({ engine, workspaceRoot, output: index, ui })
    } else {
      await syncConfluenceOutput(deps, { output, index })
    }
  }
}

/** An output can be synced when its platform's description knows it and every setting it requires is filled in. */
function isSyncable (description: ConfigDescription, output: Output): boolean {
  const platform = description.platforms.find(each => each.id === output.platform)
  const held = output as unknown as Record<string, unknown>

  return platform !== undefined && platform.fields.every(field => field.required !== true || (typeof held[field.key] === 'string' && held[field.key] !== ''))
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
