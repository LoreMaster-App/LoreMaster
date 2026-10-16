import type { Generator } from '../engine-protocol'
import { GENERATOR_TYPES, generatorTypeOf, RECOGNISED_FROM_OUTPUT, type GeneratorType } from '../generators-view'
import { parseList } from '../storage-editing'
import type { GeneratorFormValues, GeneratorKind } from './generator-form.contract'

const RECOGNISE = 'recognise'

/** The id the form uses for a type: the type itself, or `recognise` for the typeless generator. */
export function kindOf (type: string): string {
  return type === '' ? RECOGNISE : type
}

/** The type a kind id stands for. */
export function typeOfKind (kind: string): string {
  return kind === RECOGNISE ? '' : kind
}

function toKind (type: GeneratorType): GeneratorKind {
  const readsSource = type.usesSource === true

  return {
    kind:          kindOf(type.type),
    label:         type.label,
    description:   type.description,
    defaultOutput: type.defaultOutput,
    readsSource,
    readsHint:     type.inputHint,
    readsRequired: readsSource || type.type === '',
    defaultReads:  readsSource ? 'storybook-static' : '',
    command:       type.command === 'required' ? 'required' : 'optional',
  }
}

/** Every kind the form offers when adding one. */
export function allKinds (): GeneratorKind[] {
  return [RECOGNISED_FROM_OUTPUT, ...GENERATOR_TYPES].map(type => toKind(type))
}

/** The kind of an existing generator: the only one its form offers, since a type cannot change. */
export function kindsFor (generator: Generator): GeneratorKind[] {
  return [toKind(generatorTypeOf(generator.type))]
}

/** A generator's settings as form fields. */
export function generatorToValues (generator: Generator): GeneratorFormValues {
  return {
    kind:    kindOf(generator.type),
    output:  generator.output,
    reads:   generatorTypeOf(generator.type).usesSource === true ? generator.source ?? '' : (generator.input ?? []).join(', '),
    command: generator.prepare ?? '',
    title:   generator.title ?? '',
  }
}

/** A new generator from the form's fields; an empty field is left out. */
export function valuesToGenerator (values: GeneratorFormValues): Generator {
  const type = typeOfKind(values.kind)
  const reads = values.reads.trim()
  const patterns = parseList(reads)
  const command = values.command.trim()
  const title = values.title.trim()

  return {
    type,
    output: values.output.trim().replaceAll('\\', '/'),
    ...generatorTypeOf(type).usesSource === true ? (reads === '' ? {} : { source: reads }) : (patterns.length > 0 ? { input: patterns } : {}),
    ...command !== '' && { prepare: command },
    ...title !== '' && { title },
  }
}
