import * as vscode from 'vscode'
import { parseList } from '../storage-editing'
import { pickWorkspaceFolder } from '../workspace-files'
import { outputProblem } from './generator-placement.algorithm'
import { generatorTypeOf } from './generator-type.config'
import type { GeneratorNode } from './generators-view.client'
import { updateGenerator, type GeneratorPatch } from './generators.use-case'
import type { GeneratorsViewDeps } from './add-generator.handler'

/** The command id contributed in package.json. */
export const EDIT_GENERATOR_COMMAND = 'loreMaster.editGenerator'

type Field = 'output' | 'input' | 'title' | 'prepare'

/**
 * Changes one setting of a generator: its output folder, what it reads, or the title of its
 * index page. The kind cannot change; remove the generator and add another for that. A value
 * the engine refuses shows its reason and writes nothing.
 */
export async function editGeneratorCommand (deps: GeneratorsViewDeps, node: GeneratorNode | undefined): Promise<void> {
  if (!node) {
    return
  }
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    return
  }
  const type = generatorTypeOf(node.generator.type)

  const field = await pickField(node)
  if (!field) {
    return
  }
  const patch = await askPatch(field, node, type.inputHint)
  if (!patch) {
    return
  }

  try {
    await updateGenerator(deps.engine, folder, node.index, patch)
    deps.provider.refresh()
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}

async function pickField (node: GeneratorNode): Promise<Field | undefined> {
  const { generator } = node
  const picked = await vscode.window.showQuickPick(
    [
      { label: 'Output folder', description: generator.output, field: 'output' },
      generatorTypeOf(generator.type).usesSource === true
        ? { label: 'Storybook to read', description: generator.source ?? '(not set)', field: 'input' }
        : { label: 'What it reads', description: generator.input && generator.input.length > 0 ? generator.input.join(', ') : '(the default)', field: 'input' },
      { label: 'Index page title', description: generator.title ?? '(the default)', field: 'title' },
      { label: 'Command that produces it', description: generator.prepare ?? '(none)', field: 'prepare' },
    ],
    { title: `LoreMaster: change a setting of the ${generatorTypeOf(generator.type).label} generator` },
  ) as { field: Field } | undefined

  return picked?.field
}

/** Asks for the new value; undefined means cancelled. */
async function askPatch (field: Field, node: GeneratorNode, inputHint: string): Promise<GeneratorPatch | undefined> {
  const { generator } = node
  if (field === 'output') {
    const output = await vscode.window.showInputBox({ title: 'LoreMaster: output folder', prompt: 'A folder relative to the workspace. Pages already written to the old folder stay there.', value: generator.output, validateInput: value => outputProblem(value) })

    return output === undefined ? undefined : { output: output.trim().replaceAll('\\', '/') }
  }
  if (field === 'input' && generatorTypeOf(generator.type).usesSource === true) {
    const source = await vscode.window.showInputBox({
      title:         'LoreMaster: Storybook to read',
      prompt:        inputHint,
      value:         generator.source ?? '',
      validateInput: value => value.trim() === '' ? 'Name the Storybook: a URL or a build folder.' : undefined,
    })

    return source === undefined ? undefined : { source: source.trim() }
  }
  if (field === 'input') {
    const input = await vscode.window.showInputBox({ title: 'LoreMaster: what to read', prompt: inputHint, value: (generator.input ?? []).join(', ') })

    return input === undefined ? undefined : { input: parseList(input) }
  }
  if (field === 'prepare') {
    const required = generator.type === ''
    const command = await vscode.window.showInputBox({
      title:         'LoreMaster: command that produces what it reads',
      prompt:        required ? 'A shell command run from the workspace. Required: this generator has nothing else to read.' : 'A shell command run before generating; leave empty for none.',
      value:         generator.prepare ?? '',
      validateInput: value => required && value.trim() === '' ? 'Name the command that writes the files to read.' : undefined,
    })

    return command === undefined ? undefined : { prepare: command.trim() }
  }
  const title = await vscode.window.showInputBox({ title: 'LoreMaster: index page title', prompt: 'Leave empty for the default.', value: generator.title ?? '' })

  return title === undefined ? undefined : { title: title.trim() }
}
