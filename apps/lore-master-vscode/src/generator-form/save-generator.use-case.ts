import type { EngineClient } from '../engine-process'
import type { Generator } from '../engine-protocol'
import { addGenerator, isFolderSynced, readGeneratorSettings, syncedRoots, updateGenerator } from '../generators-view'
import { parseList } from '../storage-editing'
import type { GeneratorFormValues } from './generator-form.contract'
import { allKinds, kindsFor, valuesToGenerator } from './generator-form-values.mapper'
import { type GeneratorFormErrors, validateGeneratorForm } from './generator-form.validator'

export interface SaveGeneratorDeps {
  engine:        EngineClient
  workspaceRoot: string
  /** The generator being edited; absent when adding one. */
  index?:        number
}

export type SaveGeneratorResult =
  | { saved: true, index: number, generator: Generator, notSynced?: string } |
  { saved: false, errors: GeneratorFormErrors, message?: string }

/**
 * Validates the form and writes the generator: appended when adding, changed in place when
 * editing (its kind cannot change). A value the engine refuses comes back as the message and
 * nothing is written. `notSynced` says the output folder is outside every folder the storages
 * sync, so those pages would never leave the workspace.
 */
export async function saveGenerator (deps: SaveGeneratorDeps, values: GeneratorFormValues): Promise<SaveGeneratorResult> {
  try {
    const settings = await readGeneratorSettings(deps.engine, deps.workspaceRoot)
    const editing = deps.index === undefined ? undefined : settings.generators[deps.index]
    if (!editing && deps.index !== undefined) {
      return { saved: false, errors: {}, message: 'That generator is no longer in your LoreMaster settings.' }
    }
    const kinds = editing ? kindsFor(editing) : allKinds()
    const others = settings.generators.filter((_, position) => position !== deps.index).map(generator => generator.output)
    const errors = validateGeneratorForm(values, kinds, others)
    if (Object.keys(errors).length > 0) {
      return { saved: false, errors }
    }

    const generator = valuesToGenerator(values)
    let index: number
    if (deps.index === undefined) {
      index = await addGenerator(deps.engine, deps.workspaceRoot, generator)
    } else {
      index = deps.index
      await updateGenerator(deps.engine, deps.workspaceRoot, index, {
        output:  generator.output,
        input:   generator.source === undefined ? parseList(values.reads) : [],
        source:  generator.source ?? '',
        title:   generator.title ?? '',
        prepare: generator.prepare ?? '',
      })
    }
    const notSynced = isFolderSynced(generator.output, settings.outputs)
      ? undefined
      : `${generator.output} is not inside a folder any storage syncs (${syncedRoots(settings.outputs).join(', ') || 'none'}), so these pages would not reach the platform; add it with Edit storage → Folders to sync.`

    return { saved: true, index, generator, notSynced }
  } catch (error) {
    return { saved: false, errors: {}, message: error instanceof Error ? error.message : String(error) }
  }
}
