import * as vscode from 'vscode'
import type { GeneratorNode, GeneratorsViewDeps } from '../generators-view'
import { ADD_GENERATOR_COMMAND, addGeneratorForm, EDIT_GENERATOR_COMMAND, editGeneratorForm } from './open-generator-form.handler'

/** Registers the commands that open the generator form: add one, or edit the row's generator. */
export function registerGeneratorForm (deps: GeneratorsViewDeps): vscode.Disposable {
  return vscode.Disposable.from(
    vscode.commands.registerCommand(ADD_GENERATOR_COMMAND, () => addGeneratorForm(deps)),
    vscode.commands.registerCommand(EDIT_GENERATOR_COMMAND, (node: GeneratorNode | undefined) => editGeneratorForm(deps, node)),
  )
}
