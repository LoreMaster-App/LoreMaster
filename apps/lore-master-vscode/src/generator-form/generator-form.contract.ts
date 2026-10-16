/** Every field of a generator as the form holds it: all text, so an empty field is simply ''. */
export interface GeneratorFormValues {
  /** Which kind of generator: its type, `recognise` for one identified from its output, '' while unchosen. */
  kind:    string
  /** The folder the pages go to. */
  output:  string
  /** What it reads: a Storybook URL or folder, or file patterns separated by commas. */
  reads:   string
  /** The command that produces what it reads (`prepare`). */
  command: string
  /** The title of the index page; '' is the default. */
  title:   string
}

/** A kind of generator as the form offers it. */
export interface GeneratorKind {
  kind:          string
  label:         string
  description:   string
  defaultOutput: string
  /** Whether it reads one `source` (a URL or a build folder) or a set of file patterns. */
  readsSource:   boolean
  readsHint:     string
  /** Whether the reads must be given. */
  readsRequired: boolean
  /** What to put in the reads field when the kind is chosen. */
  defaultReads:  string
  command:       'optional' | 'required'
}

/** What the form is opened with. */
export interface GeneratorFormInit {
  mode:         'create' | 'edit'
  values:       GeneratorFormValues
  kinds:        GeneratorKind[]
  /** The output folders other generators already write to. */
  takenOutputs: string[]
}

/** The form's messages to the extension. */
export type GeneratorFormRequest =
  | { type: 'ready' } |
  { type: 'save', values: GeneratorFormValues } |
  { type: 'cancel' }

/** The extension's messages to the form. */
export type GeneratorFormReply =
  | { type: 'init', init: GeneratorFormInit } |
  { type: 'invalid', errors: Partial<Record<keyof GeneratorFormValues, string>>, message?: string }
