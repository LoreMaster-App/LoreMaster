import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { Generator } from '../engine-protocol'
import { GENERATE_AND_SYNC_COMMAND } from '../generate-and-sync'
import { runAndReport } from '../generators-command'
import { parseList } from '../storage-editing'
import { pickWorkspaceFolder } from '../workspace-files'
import { freeOutputFolder, isFolderSynced, outputProblem, syncedRoots } from './generator-placement.algorithm'
import { GENERATOR_TYPES, type GeneratorType, RECOGNISED_FROM_OUTPUT } from './generator-type.config'
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
const GENERATE_AND_SYNC = 'Generate and sync'

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
    const input = await vscode.window.showInputBox({
      title:         `LoreMaster: what should ${type.label} read?`,
      prompt:        type.inputHint,
      value:         type.usesSource === true ? 'storybook-static' : '',
      validateInput: value => value.trim() === '' && (type.usesSource === true || type.type === '')
        ? 'This one needs something to read: a Storybook URL or build folder, or the files the command writes.'
        : undefined,
    })
    if (input === undefined) {
      return
    }

    const patterns = parseList(input)
    const reads = type.usesSource === true ? { source: input.trim() } : (patterns.length > 0 ? { input: patterns } : {})
    const command = type.command === undefined ? undefined : await askCommand(type)
    if (command === null) {
      return
    }
    const generator: Generator = {
      type:   type.type,
      output: output.trim().replaceAll('\\', '/'),
      ...reads,
      ...command && { prepare: command },
    }
    const index = await addGenerator(deps.engine, folder, generator)
    deps.provider.refresh()

    const warning = isFolderSynced(generator.output, settings.outputs)
      ? ''
      : ` ${generator.output} is not inside a folder any storage syncs (${syncedRoots(settings.outputs).join(', ') || 'none'}), so these pages would not reach the platform; add it with Edit storage → Folders to sync.`
    const answer = await vscode.window.showInformationMessage(`LoreMaster: added the ${type.label} generator → ${generator.output}.${warning}`, RUN_NOW, GENERATE_AND_SYNC)
    if (answer === RUN_NOW) {
      deps.provider.recordRun(await runAndReport(deps, folder, [index]))
    } else if (answer === GENERATE_AND_SYNC) {
      await vscode.commands.executeCommand(GENERATE_AND_SYNC_COMMAND)
      deps.provider.refresh()
    }
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}

/**
 * Asks for the command that produces what the generator reads. Undefined means none was given
 * (allowed when optional); null means the user cancelled.
 */
async function askCommand (type: GeneratorType): Promise<string | undefined | null> {
  const required = type.command === 'required'
  const command = await vscode.window.showInputBox({
    title:  `LoreMaster: which command produces what ${type.label} reads?`,
    prompt: required
      ? 'A shell command run from the workspace (a test run, a build, a script), e.g. npm test -- --reporters=jest-junit. It runs when you generate, after you approve it.'
      : 'Optional. A shell command that builds what it reads, e.g. npm run build-storybook. Leave empty to build it yourself.',
    validateInput: value => required && value.trim() === '' ? 'Name the command that writes the files to read.' : undefined,
  })
  if (command === undefined) {
    return null
  }

  return command.trim() === '' ? undefined : command.trim()
}

async function pickType (): Promise<GeneratorType | undefined> {
  const picked = await vscode.window.showQuickPick(
    [RECOGNISED_FROM_OUTPUT, ...GENERATOR_TYPES].map(type => ({ label: type.label, description: type.defaultOutput, detail: type.description, type })),
    { title: 'LoreMaster: add a generator', placeHolder: 'What should be written as pages?' },
  ) as { type: GeneratorType } | undefined

  return picked?.type
}
