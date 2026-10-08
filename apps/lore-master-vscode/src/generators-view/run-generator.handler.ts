import * as vscode from 'vscode'
import { runAndReport } from '../generators-command'
import { pickWorkspaceFolder } from '../workspace-files'
import type { GeneratorsViewDeps } from './add-generator.handler'
import type { GeneratorNode } from './generators-view.client'

/** The command id contributed in package.json. */
export const RUN_GENERATOR_COMMAND = 'loreMaster.runGenerator'

/** Runs the generator a row stands for, reports the summary, and shows it on the row. */
export async function runGeneratorCommand (deps: GeneratorsViewDeps, node: GeneratorNode | undefined): Promise<void> {
  if (!node) {
    return
  }
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    return
  }

  try {
    deps.provider.recordRun(await runAndReport(deps, folder, [node.index]))
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}
