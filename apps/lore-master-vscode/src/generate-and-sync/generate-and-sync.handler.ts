import * as vscode from 'vscode'
import { listGenerators, runAndReport } from '../generators-command'
import { sync, type SyncCommandDeps } from '../sync-command'
import { pickWorkspaceFolder } from '../workspace-files'
import { confirmationAfterFailures, failedRuns } from './generation-failures.policy'

/** The command id contributed in package.json. */
export const GENERATE_AND_SYNC_COMMAND = 'loreMaster.generateAndSync'

/** The setting that makes the ordinary Sync generate first. */
export const GENERATE_BEFORE_SYNC_SETTING = 'generateBeforeSync'

export interface GenerateAndSyncDeps extends SyncCommandDeps {
  /** How the sync itself runs; the real Sync command unless a test supplies one. */
  runSync?: (deps: SyncCommandDeps, folder?: string) => Promise<void>
}

const SYNC_ANYWAY = 'Sync anyway'

/**
 * Runs the workspace's generators, then syncs, so generated pages (test results, API
 * documentation) are as fresh as the code when they reach the storage. The sync previews its plan
 * as usual. When a generator fails the user is asked before the pages go out; warnings alone do
 * not stop it. With no generators configured it says so and syncs the Markdown.
 */
export async function generateAndSyncCommand (deps: GenerateAndSyncDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to generate and sync its documentation.')

    return
  }

  try {
    const configured = await listGenerators(deps.engine, folder)
    if (configured.length === 0) {
      await vscode.window.showInformationMessage('LoreMaster: no generators are configured, so only the Markdown is synced. Add one from the Generators view.')
    } else if (!await generate(deps, folder)) {
      return
    }
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)

    return
  }

  await (deps.runSync ?? sync)(deps, folder)
}

/** The Sync command: it generates first when the user asked it to in the settings. */
export async function syncCommand (deps: GenerateAndSyncDeps): Promise<void> {
  if (vscode.workspace.getConfiguration('loreMaster').get<boolean>(GENERATE_BEFORE_SYNC_SETTING, false)) {
    await generateAndSyncCommand(deps)

    return
  }

  await (deps.runSync ?? sync)(deps)
}

/** Runs the generators and reports; false when some failed and the user chose not to sync. */
async function generate (deps: GenerateAndSyncDeps, folder: string): Promise<boolean> {
  const failed = failedRuns(await runAndReport(deps, folder))
  if (failed.length === 0) {
    return true
  }

  return await vscode.window.showWarningMessage(confirmationAfterFailures(failed), { modal: true }, SYNC_ANYWAY) === SYNC_ANYWAY
}
