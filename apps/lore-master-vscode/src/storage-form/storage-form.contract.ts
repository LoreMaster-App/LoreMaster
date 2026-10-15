import type { Page, Space } from '../engine-protocol'

/** The platforms a storage can publish to, as the form offers them. */
export type FormPlatform = 'confluence' | 'github-pages' | 'github-wiki'

/** Every field of a storage as the form holds it: all text, so an empty field is simply ''. */
export interface StorageFormValues {
  /** '' while the user has not chosen one (a new storage has none preselected). */
  platform:       FormPlatform | ''
  /** The Confluence site's base URL (a stored connection). */
  connection:     string
  space:          string
  parentPageId:   string
  titlePrefix:    string
  direction:      string
  mermaidMode:    string
  linkMode:       string
  titleCollision: string
  repo:           string
  branch:         string
  path:           string
  /** The page template: a built-in's name, a path or a Confluence page; '' is the default. */
  template:       string
  /** Folders to sync, separated by commas. */
  roots:          string
  /** Paths this storage leaves out, separated by commas. */
  excludes:       string
  /** Paths every storage leaves out (a setting of the whole workspace), separated by commas. */
  ignore:         string
}

/** A stored connection the form can pick. */
export interface ConnectionChoice {
  baseUrl: string
  label:   string
}

/** What the form is opened with. */
export interface StorageFormInit {
  mode:        'create' | 'edit'
  values:      StorageFormValues
  connections: ConnectionChoice[]
}

/** The form's messages to the extension. */
export type FormRequest =
  | { type: 'ready' } |
  { type: 'listSpaces', connection: string } |
  { type: 'searchPages', connection: string, space: string, query: string } |
  { type: 'addConnection' } |
  { type: 'save', values: StorageFormValues } |
  { type: 'cancel' }

/** The extension's messages to the form. */
export type FormReply =
  | { type: 'init', init: StorageFormInit } |
  { type: 'spaces', spaces: Space[] } |
  { type: 'pages', pages: Page[] } |
  { type: 'connections', connections: ConnectionChoice[], selected: string } |
  { type: 'lookupError', message: string } |
  { type: 'invalid', errors: Partial<Record<keyof StorageFormValues, string>>, message?: string } |
  { type: 'saved' }
