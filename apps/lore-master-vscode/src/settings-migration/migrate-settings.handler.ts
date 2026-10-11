import * as vscode from 'vscode'
import { migrateSettings, type MigrationEngine } from './migrate-settings.use-case'

/** Opens VS Code's Settings editor filtered to this extension's settings. */
export const OPEN_SETTINGS_QUERY = '@ext:LoreMaster.loremaster'

/**
 * Runs once per activation: moves a workspace's .lore-master.yaml into its VS Code settings
 * and tells the user, once, where the configuration went. The yaml is left in place (it is
 * not read again once the settings exist); the message says it can be deleted.
 */
export async function runSettingsMigration (deps: { engine: MigrationEngine; onMigrated: () => void }): Promise<void> {
  const workspaceRoots = (vscode.workspace.workspaceFolders ?? []).map(folder => folder.uri.fsPath)
  if (workspaceRoots.length === 0) {
    return
  }

  const outcome = await migrateSettings({ engine: deps.engine, workspaceRoots })
  if (outcome.migrated.length > 0) {
    deps.onMigrated()
    const choice = await vscode.window.showInformationMessage(
      'LoreMaster now keeps its configuration in your VS Code settings (.vscode/settings.json). Your .lore-master.yaml was imported and is no longer read; you can delete it.',
      'Open settings',
    )
    if (choice === 'Open settings') {
      await vscode.commands.executeCommand('workbench.action.openSettings', OPEN_SETTINGS_QUERY)
    }
  }
  for (const failure of outcome.failures) {
    await vscode.window.showWarningMessage(`LoreMaster could not import .lore-master.yaml into your settings: ${failure}`)
  }
}
