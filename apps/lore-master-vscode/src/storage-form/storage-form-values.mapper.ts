import type { Content, Output, Settings } from '../engine-protocol'
import { parseList } from '../storage-editing'
import type { FormPlatform, StorageFormValues } from './storage-form.contract'

/** An empty form: no platform chosen, the defaults a new Confluence storage starts from. */
export function emptyValues (): StorageFormValues {
  return {
    name:           '',
    platform:       '',
    connection:     '',
    space:          '',
    parentPageId:   '',
    titlePrefix:    '',
    direction:      'to-platform',
    mermaidMode:    'image',
    linkMode:       'title',
    titleCollision: 'fail',
    repo:           '',
    branch:         '',
    path:           '',
    template:       '',
    roots:          '.',
    excludes:       '',
    ignore:         '',
  }
}

function firstMarkdown (output: Output): Content | undefined {
  return output.content.find(content => content.type === 'markdown')
}

/** The form's view of an existing storage and of the workspace-wide ignore list. */
export function outputToValues (output: Output, settings: Settings): StorageFormValues {
  const content = firstMarkdown(output)
  const platform: FormPlatform = output.platform === 'github-pages' || output.platform === 'github-wiki' ? output.platform : 'confluence'

  return {
    name:           output.name ?? '',
    platform,
    connection:     output.baseUrl,
    space:          output.space,
    parentPageId:   output.parentPageId,
    titlePrefix:    output.titlePrefix,
    direction:      output.direction || 'to-platform',
    mermaidMode:    output.mermaidMode || 'image',
    linkMode:       output.linkMode || 'title',
    titleCollision: output.titleCollision || 'fail',
    repo:           output.repo ?? '',
    branch:         output.branch ?? '',
    path:           output.path ?? '',
    template:       output.template ?? '',
    roots:          (content?.roots ?? ['.']).join(', '),
    excludes:       (content?.excludes ?? []).join(', '),
    ignore:         (settings.ignore ?? []).join(', '),
  }
}

function blankOutput (platform: FormPlatform): Output {
  return {
    platform,
    baseUrl:        '',
    space:          '',
    parentPageId:   '',
    titlePrefix:    '',
    direction:      'to-platform',
    content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
    mermaidMode:    '',
    titleCollision: '',
    linkMode:       '',
  }
}

/**
 * The storage the values describe. A storage being edited is the base, so what the form does
 * not show (extra content entries, templates, `include`) is kept; a new one starts blank.
 * Fields that do not apply to the platform are left empty.
 */
export function valuesToOutput (values: StorageFormValues, base?: Output): Output {
  const platform = values.platform === '' ? 'confluence' : values.platform
  const output: Output = { ...(base ?? blankOutput(platform)), platform }

  const roots = parseList(values.roots)
  const excludes = parseList(values.excludes)
  const markdown: Content = { ...(firstMarkdown(output) ?? { type: 'markdown', template: 'default', roots: ['.'] }), roots: roots.length > 0 ? roots : ['.'], excludes }
  const hadMarkdown = firstMarkdown(output) !== undefined
  output.content = hadMarkdown
    ? output.content.map(content => content === firstMarkdown(output) ? markdown : content)
    : [markdown, ...output.content]

  if (platform === 'confluence') {
    output.baseUrl = values.connection.trim()
    output.space = values.space.trim()
    output.parentPageId = values.parentPageId.trim()
    output.titlePrefix = values.titlePrefix.trim()
    output.direction = values.direction
    output.mermaidMode = values.mermaidMode
    output.linkMode = values.linkMode
    output.titleCollision = values.titleCollision
    output.repo = undefined
    output.branch = undefined
    output.path = undefined
  } else {
    output.baseUrl = ''
    output.space = ''
    output.parentPageId = ''
    output.titlePrefix = ''
    output.repo = values.repo.trim()
    output.branch = values.branch.trim()
    output.path = platform === 'github-pages' && values.path.trim() !== '' ? values.path.trim() : undefined
  }

  const name = values.name.trim()
  output.name = name === '' ? undefined : name
  const template = values.template.trim()
  output.template = template === '' || template === 'default' ? undefined : template

  return output
}

/**
 * A name for a storage that was given none: where it goes, numbered when another storage already
 * has that name, so two storages on one platform are still told apart.
 */
export function suggestStorageName (values: StorageFormValues, taken: string[]): string {
  const lower = new Set(taken.map(name => name.toLowerCase()))
  let base = 'Confluence'
  if (values.platform === 'github-pages') {
    base = `GitHub Pages · ${values.repo.trim() || values.branch.trim() || 'this repository'}`
  } else if (values.platform === 'github-wiki') {
    base = `GitHub Wiki · ${values.repo.trim() || 'this repository'}`
  } else if (values.space.trim() !== '') {
    base = `Confluence · ${values.space.trim()}`
  }
  let candidate = base
  for (let number = 2; lower.has(candidate.toLowerCase()); number++) {
    candidate = `${base} (${number})`
  }

  return candidate
}

/** The workspace-wide ignore list the values carry, or undefined for none. */
export function ignoreFromValues (values: StorageFormValues): string[] | undefined {
  const patterns = parseList(values.ignore)

  return patterns.length > 0 ? patterns : undefined
}
