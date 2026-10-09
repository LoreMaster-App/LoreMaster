import type { Content, Output } from '../engine-protocol'

/** How a setting is asked for: a pick from allowed values, a line of text, or a list. */
export type FieldKind = 'enum' | 'text' | 'list'

/** One setting of a storage the sidebar can change. */
export interface StorageField {
  key:      string
  label:    string
  kind:     FieldKind
  /** The allowed values of an enum. */
  options?: readonly string[]
  /** What to tell the user when asking for text or a list. */
  hint?:    string
  /** The current value as the user would type it ('' when unset). */
  read (output: Output): string
  /** The output with the value applied. Validation is the engine's, on save. */
  write (output: Output, value: string): Output
}

/** A comma- or line-separated list, trimmed, without empty entries. */
export function parseList (value: string): string[] {
  return value.split(/[,\n]/).map(entry => entry.trim()).filter(entry => entry !== '')
}

function firstMarkdown (output: Output): Content | undefined {
  return output.content.find(content => content.type === 'markdown')
}

/** Applies a change to the output's first Markdown content entry (the one a workspace has). */
function withContent (output: Output, change: (content: Content) => Content): Output {
  const index = output.content.findIndex(content => content.type === 'markdown')
  if (index === -1) {
    return output
  }

  return { ...output, content: output.content.map((content, position) => position === index ? change(content) : content) }
}

const enumField = (key: 'direction' | 'mermaidMode' | 'linkMode' | 'titleCollision', label: string, options: readonly string[]): StorageField => ({
  key,
  label,
  kind:  'enum',
  options,
  read:  output => output[key],
  write: (output, value) => ({ ...output, [key]: value }),
})

const textField = (key: 'titlePrefix' | 'repo' | 'branch' | 'path', label: string, hint: string): StorageField => ({
  key,
  label,
  kind:  'text',
  hint,
  read:  output => output[key] ?? '',
  write: (output, value) => ({ ...output, [key]: value.trim() }),
})

const rootsField: StorageField = {
  key:   'roots',
  label: 'Folders to sync',
  kind:  'list',
  hint:  'Folders, relative to the workspace and separated by commas. Leave empty for the whole workspace.',
  read:  output => (firstMarkdown(output)?.roots ?? []).join(', '),
  write: (output, value) => withContent(output, content => {
    const roots = parseList(value)

    return { ...content, roots: roots.length > 0 ? roots : ['.'] }
  }),
}

const excludesField: StorageField = {
  key:   'excludes',
  label: 'Excluded paths',
  kind:  'list',
  hint:  'Gitignore-style patterns, separated by commas, that this storage leaves out. Leave empty for none.',
  read:  output => (firstMarkdown(output)?.excludes ?? []).join(', '),
  write: (output, value) => withContent(output, content => ({ ...content, excludes: parseList(value) })),
}

const CONFLUENCE_FIELDS: readonly StorageField[] = [
  textField('titlePrefix', 'Title prefix', 'Pages are titled "<prefix>: <first heading>".'),
  enumField('direction', 'Direction', ['to-platform', 'two-way']),
  enumField('mermaidMode', 'Mermaid diagrams', ['image', 'code']),
  enumField('linkMode', 'Links between pages', ['title', 'id']),
  enumField('titleCollision', 'When a page title already exists', ['fail', 'adopt']),
  rootsField,
  excludesField,
]

const GITHUB_PAGES_FIELDS: readonly StorageField[] = [
  textField('repo', 'Repository', 'owner/name or a clone URL. Leave empty for this workspace\'s own origin.'),
  textField('branch', 'Branch', 'The branch the site is published to. Leave empty for gh-pages.'),
  textField('path', 'Folder in the branch', 'Publish into this folder and leave the rest of the branch alone, so another site can share it. Leave empty to own the whole branch.'),
  rootsField,
  excludesField,
]

/** The settings that can be changed for a storage of this platform. */
export function fieldsFor (output: Output): readonly StorageField[] {
  return output.platform === 'github-pages' ? GITHUB_PAGES_FIELDS : CONFLUENCE_FIELDS
}
