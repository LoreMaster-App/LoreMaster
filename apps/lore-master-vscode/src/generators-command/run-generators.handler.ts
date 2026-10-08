import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { Generator } from '../engine-protocol'
import { pickWorkspaceFolder } from '../workspace-files'
import { summariseRun } from './generator-run-report.mapper'
import { listGenerators, runGenerators } from './run-generators.use-case'

/** The command id contributed in package.json. */
export const RUN_GENERATORS_COMMAND = 'loreMaster.runGenerators'

/** What the command is given at registration. */
export interface RunGeneratorsDeps {
  engine: EngineClient
  output: vscode.OutputChannel
}

/**
 * Runs the workspace's generators: test results, Go package documentation, OpenAPI
 * descriptions. They write ordinary Markdown into the folders `.lore-master.yaml` names, which
 * the next sync publishes like any other page. With several generators it asks which; the
 * answer is a one-line summary, with the details and any problems in the LoreMaster output.
 */
export async function runGeneratorsCommand (deps: RunGeneratorsDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to run its generators.')

    return
  }

  try {
    const configured = await listGenerators(deps.engine, folder)
    if (configured.length === 0) {
      await vscode.window.showInformationMessage('LoreMaster: no generators are configured. Add a "generators:" list to .lore-master.yaml (see docs/generators.md).')

      return
    }
    const chosen = await chooseGenerators(configured)
    if (chosen === undefined) {
      return
    }

    const result = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: 'LoreMaster: running generators' },
      () => runGenerators(deps.engine, folder, chosen),
    )
    const report = summariseRun(result)
    for (const line of report.lines) {
      deps.output.appendLine(line)
    }
    if (report.hasProblems) {
      deps.output.show(true)
      await vscode.window.showWarningMessage(`${report.headline} Some need a look; see the LoreMaster output.`)
    } else {
      await vscode.window.showInformationMessage(report.headline)
    }
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}

/** Which generators to run: all of them (`[]` is never sent; `undefined` means cancelled). */
async function chooseGenerators (configured: Generator[]): Promise<number[] | undefined> {
  if (configured.length === 1) {
    return [0]
  }
  const picked = await vscode.window.showQuickPick(
    [
      { label: 'All generators', description: `${configured.length} configured`, indexes: configured.map((_, index) => index) },
      ...configured.map((generator, index) => ({ label: generator.type, description: `→ ${generator.output}`, indexes: [index] })),
    ],
    { title: 'LoreMaster: run which generators?' },
  ) as { indexes: number[] } | undefined

  return picked?.indexes
}
