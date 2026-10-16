import { outputProblem } from '../generators-view'
import type { GeneratorFormValues, GeneratorKind } from './generator-form.contract'

export type GeneratorFormErrors = Partial<Record<keyof GeneratorFormValues, string>>

/**
 * What is wrong with the form's values, by field; empty when it can be saved. The engine
 * validates again on save, so this only catches what the person can fix on the spot.
 */
export function validateGeneratorForm (values: GeneratorFormValues, kinds: readonly GeneratorKind[], otherOutputs: readonly string[]): GeneratorFormErrors {
  const errors: GeneratorFormErrors = {}
  const kind = kinds.find(each => each.kind === values.kind)
  if (!kind) {
    errors.kind = 'Choose what should be written as pages.'

    return errors
  }

  const problem = outputProblem(values.output)
  const folder = values.output.trim().replaceAll('\\', '/').replace(/\/+$/, '')
  if (problem) {
    errors.output = problem
  } else if (otherOutputs.some(other => other.replace(/\/+$/, '') === folder)) {
    errors.output = `Another generator already writes to ${folder}. Each generator needs a folder of its own.`
  }

  if (kind.readsRequired && values.reads.trim() === '') {
    errors.reads = kind.readsSource
      ? 'Name the Storybook: a URL or a build folder.'
      : 'Name what to read: the files or folder the command writes.'
  }
  if (kind.command === 'required' && values.command.trim() === '') {
    errors.command = 'Name the command that writes the files to read.'
  }

  return errors
}
