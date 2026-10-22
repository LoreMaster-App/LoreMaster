import { allOutputKeys, blankOutput, fieldsOf } from '../config-description'
import type { ConfigDescription, Content, FieldDescriptor, Output, PlatformDescriptor, Settings } from '../engine-protocol'
import { parseList } from '../storage-editing'
import type { StorageFormDescription, StorageFormValues } from './storage-form.contract'

type Record2 = Record<string, unknown>

/** What the form shows, taken from the engine's description. */
export function formDescription (description: ConfigDescription): StorageFormDescription {
  const project = description.sections.find(section => section.id === 'project')

  return {
    platforms:     description.platforms,
    outputFields:  description.outputFields,
    contentFields: description.contentFields,
    ignoreField:   project?.settings.find(field => field.key === 'ignore'),
  }
}

/** The value a field starts with in a new storage: a text field starts empty (its default is a placeholder), the rest on their default. */
function initialValue (field: FieldDescriptor): string {
  return field.type === 'text' ? '' : field.default ?? ''
}

/** An empty form: no platform chosen, every choice on its default. */
export function emptyValues (description: ConfigDescription): StorageFormValues {
  const values: StorageFormValues = { platform: '', ignore: '' }
  for (const platform of description.platforms) {
    for (const field of platform.fields) {
      values[field.key] ||= initialValue(field)
    }
  }
  for (const field of [...description.outputFields, ...description.contentFields]) {
    values[field.key] = initialValue(field)
  }

  return values
}

function firstMarkdown (output: Output): Content | undefined {
  return output.content.find(content => content.type === 'markdown')
}

function text (value: unknown): string {
  if (Array.isArray(value)) {
    return value.join(', ')
  }
  if (typeof value === 'number') {
    return String(value)
  }

  return typeof value === 'string' ? value : ''
}

/** The form's view of an existing storage and of the workspace-wide ignore list. */
export function outputToValues (description: ConfigDescription, output: Output, settings: Settings): StorageFormValues {
  const values = emptyValues(description)
  const platform = description.platforms.find(each => each.id === output.platform)
  values.platform = platform?.id ?? ''
  const held = output as unknown as Record2
  const shown = platform ? fieldsOf(description, platform) : description.outputFields
  for (const field of shown) {
    const shown = text(held[field.key])
    values[field.key] = shown === '' ? initialValue(field) : shown
  }
  const content = firstMarkdown(output)
  values.roots = (content?.roots ?? ['.']).join(', ')
  values.excludes = (content?.excludes ?? []).join(', ')
  values.ignore = (settings.ignore ?? []).join(', ')

  return values
}

/** Stores one field's value in the output; a list field holds a list, anything else its trimmed text. */
function assign (output: Record2, field: FieldDescriptor, value: string): void {
  if (field.type === 'list') {
    const items = parseList(value)
    if (items.length > 0) {
      output[field.key] = items
    } else {
      delete output[field.key]
    }

    return
  }
  if (field.type === 'number') {
    const typed = value.trim()
    const number = typed === '' ? NaN : Number(typed)
    if (Number.isSafeInteger(number)) {
      output[field.key] = number
    } else {
      delete output[field.key]
    }

    return
  }
  output[field.key] = value.trim()
}

/** Empties a setting that belongs to another platform than the chosen one. */
function clear (output: Record2, key: string, description: ConfigDescription): void {
  const field = [...description.outputFields, ...description.platforms.flatMap(platform => platform.fields)].find(each => each.key === key)
  if (field?.type === 'list' || field?.type === 'number') {
    delete output[key]
  } else {
    // A choice goes back to its default: the engine wants a valid direction even on a site.
    output[key] = field?.type === 'choice' ? field.default ?? '' : ''
  }
}

/**
 * The storage the values describe. A storage being edited is the base, so what the form does not
 * show (extra content entries, `include`) is kept; a new one starts blank. Settings of the other
 * platforms are emptied, so a storage moved to another platform carries nothing of the old one.
 */
export function valuesToOutput (description: ConfigDescription, values: StorageFormValues, base?: Output): Output {
  const platform: PlatformDescriptor | undefined = description.platforms.find(each => each.id === values.platform) ?? description.platforms[0]
  const output: Output = { ...(base ?? blankOutput(description, platform)), platform: platform.id }
  const held = output as unknown as Record2

  const shown = fieldsOf(description, platform)
  for (const key of allOutputKeys(description)) {
    if (shown.every(field => field.key !== key)) {
      clear(held, key, description)
    }
  }
  for (const field of shown) {
    assign(held, field, values[field.key] ?? '')
  }
  if (typeof held.name === 'string' && held.name === '') {
    delete held.name
  }

  const roots = parseList(values.roots ?? '')
  const markdown: Content = { ...(firstMarkdown(output) ?? { type: 'markdown', template: 'default', roots: ['.'] }), roots: roots.length > 0 ? roots : ['.'], excludes: parseList(values.excludes ?? '') }
  const hadMarkdown = firstMarkdown(output) !== undefined
  output.content = hadMarkdown
    ? output.content.map(content => content === firstMarkdown(output) ? markdown : content)
    : [markdown, ...output.content]

  return output
}

/** The workspace-wide ignore list the values carry, or undefined for none. */
export function ignoreFromValues (values: StorageFormValues): string[] | undefined {
  const patterns = parseList(values.ignore ?? '')

  return patterns.length > 0 ? patterns : undefined
}
