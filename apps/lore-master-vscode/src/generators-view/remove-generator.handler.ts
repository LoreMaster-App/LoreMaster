import * as vscode from 'vscode'
import { pickWorkspaceFolder } from '../workspace-files'
import type { GeneratorsViewDeps } from './add-generator.handler'
import { generatorTypeOf } from './generator-type.config'
import type { GeneratorNode } from './generators-view.client'
import { removeGenerator } from './generators.use-case'

/** The command id contributed in package.json. */
export const REMOVE_GENERATOR_COMMAND = 'loreMaster.removeGenerator'

/** Removes a generator from .lore-master.yaml after confirming. The pages it already wrote are
 *  left where they are: they are ordinary files now, and deleting the folder is the user's call. */
export async function removeGeneratorCommand (deps: GeneratorsViewDeps, node: GeneratorNode | undefined): Promise<void> {
  if (!node) {
    return
  }
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    return
  }

  const label = generatorTypeOf(node.generator.type).label
  const confirmed = await vscode.window.showWarningMessage(
    `Remove the ${label} generator? The pages it wrote to ${node.generator.output} stay; delete that folder to remove them.`,
    { modal: true },
    'Remove',
  )
  if (confirmed !== 'Remove') {
    return
  }

  try {
    await removeGenerator(deps.engine, folder, node.index)
    deps.provider.refresh()
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}
