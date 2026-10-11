import { SETTINGS_MIGRATE_METHOD, type SettingsMigrateResult } from '../engine-protocol'

/** The minimal engine surface the migration needs. */
export interface MigrationEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

export interface MigrationOutcome {
  /** The settings files written, one per folder whose .lore-master.yaml was imported. */
  migrated: string[]
  /** One message per folder whose configuration could not be read or written. */
  failures: string[]
}

/**
 * Imports each workspace folder's .lore-master.yaml into its .vscode/settings.json, once:
 * the engine does nothing for a folder that is already configured in the editor's settings
 * or has no yaml. A folder that fails does not stop the others.
 */
export async function migrateSettings (deps: { engine: MigrationEngine; workspaceRoots: readonly string[] }): Promise<MigrationOutcome> {
  const outcome: MigrationOutcome = { migrated: [], failures: [] }
  for (const workspaceRoot of deps.workspaceRoots) {
    try {
      const result = await deps.engine.request<SettingsMigrateResult>(SETTINGS_MIGRATE_METHOD, { workspaceRoot })
      if (result.migrated && result.path) {
        outcome.migrated.push(result.path)
      }
    } catch (error) {
      outcome.failures.push(error instanceof Error ? error.message : String(error))
    }
  }

  return outcome
}
