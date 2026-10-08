import {
  type Generator,
  type Output,
  SETTINGS_READ_METHOD,
  SETTINGS_SAVE_METHOD,
  type Settings,
  type SettingsReadResult,
} from '../engine-protocol'

/** The minimal engine surface the view needs. */
export interface GeneratorsViewEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** What the view shows: the generators, and the storages they should write into the reach of. */
export interface GeneratorSettings {
  generators: Generator[]
  outputs:    Output[]
}

export async function readGeneratorSettings (engine: GeneratorsViewEngine, workspaceRoot: string): Promise<GeneratorSettings> {
  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })

  return { generators: read.settings.generators ?? [], outputs: read.settings.outputs }
}

/** What can change about a generator once it exists. A cleared value is sent as absent, and
 *  the engine then removes it from the file. */
export interface GeneratorPatch {
  output?: string
  input?:  string[]
  title?:  string
}

/** Reads the settings, lets `change` rewrite the generators, and saves everything else as it was.
 *  The engine validates and keeps the author's comments, so a refused value throws with its
 *  reason and nothing is written. */
async function saveGenerators (engine: GeneratorsViewEngine, workspaceRoot: string, change: (generators: Generator[]) => Generator[]): Promise<void> {
  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  const generators = change([...(read.settings.generators ?? [])])
  const settings: Settings = { ...read.settings, generators: generators.length > 0 ? generators : undefined }

  await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings })
}

/** Appends a generator and returns its index. */
export async function addGenerator (engine: GeneratorsViewEngine, workspaceRoot: string, generator: Generator): Promise<number> {
  let index = -1
  await saveGenerators(engine, workspaceRoot, generators => {
    generators.push(generator)
    index = generators.length - 1

    return generators
  })

  return index
}

/** Changes the output, input or title of the generator at `index`. */
export async function updateGenerator (engine: GeneratorsViewEngine, workspaceRoot: string, index: number, patch: GeneratorPatch): Promise<void> {
  await saveGenerators(engine, workspaceRoot, generators => {
    if (index < 0 || index >= generators.length) {
      throw new Error('That generator is no longer in .lore-master.yaml.')
    }
    const current = generators[index]
    generators[index] = {
      type:   current.type,
      output: patch.output ?? current.output,
      input:  patch.input === undefined ? current.input : (patch.input.length > 0 ? patch.input : undefined),
      title:  patch.title === undefined ? current.title : (patch.title === '' ? undefined : patch.title),
    }

    return generators
  })
}

/** Removes the generator at `index`. The pages it wrote stay where they are. */
export async function removeGenerator (engine: GeneratorsViewEngine, workspaceRoot: string, index: number): Promise<void> {
  await saveGenerators(engine, workspaceRoot, generators => {
    if (index < 0 || index >= generators.length) {
      throw new Error('That generator is no longer in .lore-master.yaml.')
    }

    return generators.filter((_, position) => position !== index)
  })
}
