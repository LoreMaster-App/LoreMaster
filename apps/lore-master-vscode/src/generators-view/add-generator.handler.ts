import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { Generator } from '../engine-protocol'
import { runAndReport } from '../generators-command'
import { parseList } from '../storage-editing'
import { pickWorkspaceFolder } from '../workspace-files'
import { freeOutputFolder, isFolderSynced, outputProblem, syncedRoots } from './generator-placement.algorithm'
import { GENERATOR_TYPES, type GeneratorType } from './generator-type.config'
import type { GeneratorsViewProvider } from './generators-view.client'
import { addGenerator, readGeneratorSettings } from './generators.use-case'

/** The command id contributed in package.json. */
export const ADD_GENERATOR_COMMAND = 'loreMaster.addGenerator'

/** What the Generators view's commands are given at registration. */
export interface GeneratorsViewDeps {
  engine:   EngineClient
  output:   vscode.OutputChannel
  provider: GeneratorsViewProvider
}

const RUN_NOW = 'Run now'

/**
 * Adds a generator to .lore-master.yaml: which kind, where its pages go, and what it reads.
 * The engine validates the result, so a refused value shows its reason and writes nothing.
 * It warns when the output folder is outside every folder the storages sync, since those
 * pages would never leave the workspace, and offers to run the new generator at once.
 */
export async function addGeneratorCommand (deps: GeneratorsViewDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to add a generator.')

    return
  }

  try {
    const type = await pickType()
    if (!type) {
      return
    }
    const settings = await readGeneratorSettings(deps.engine, folder)
    const output = await vscode.window.showInputBox({
      title:         `LoreMaster: where should the ${type.label} pages go?`,
      prompt:        'A folder for the generated pages, relative to the workspace. Only this generator writes to it.',
      value:         freeOutputFolder(type.defaultOutput, settings.generators.map(generator => generator.output)),
      validateInput: value => outputProblem(value),
    })
    if (output === undefined) {
      return
    }
    const input = await vscode.window.showInputBox({ title: `LoreMaster: what should ${type.label} read?`, prompt: type.inputHint, value: '' })
    if (input === undefined) {
      return
    }

    const patterns = parseList(input)
    const generator: Generator = { type: type.type, output: output.trim().replaceAll('\\', '/'), ...((patterns.length > 0) && { input: patterns }) }
    const index = await addGenerator(deps.engine, folder, generator)
    deps.provider.refresh()

    const warning = isFolderSynced(generator.output, settings.outputs)
      ? ''
      : ` ${generator.output} is not inside a folder any storage syncs (${syncedRoots(settings.outputs).join(', ') || 'none'}), so these pages would not reach the platform; add it with Edit storage → Folders to sync.`
    const answer = await vscode.window.showInformationMessage(`LoreMaster: added the ${type.label} generator → ${generator.output}.${warning}`, RUN_NOW)
    if (answer === RUN_NOW) {
      deps.provider.recordRun(await runAndReport(deps, folder, [index]))
    }
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}

async function pickType (): Promise<GeneratorType | undefined> {
  const picked = await vscode.window.showQuickPick(
    GENERATOR_TYPES.map(type => ({ label: type.label, description: type.defaultOutput, detail: type.description, type })),
    { title: 'LoreMaster: add a generator', placeHolder: 'What should be written as pages?' },
  ) as { type: GeneratorType } | undefined

  return picked?.type
}
