import type { FieldDescriptor, Page, PlatformDescriptor, Space } from '../engine-protocol'

/**
 * Every field of a storage as the form holds it: all text, keyed by the engine's field keys, so an
 * empty field is simply ''. `platform` is the chosen platform's id, `roots` and `excludes` are the
 * Markdown content's, and `ignore` is the project-wide ignore list; lists are comma separated.
 */
export type StorageFormValues = Record<string, string>

/** What the form shows, all of it from the engine's description. */
export interface StorageFormDescription {
  platforms:     PlatformDescriptor[]
  /** The settings every storage has. */
  outputFields:  FieldDescriptor[]
  /** The Markdown content's settings (folders, excluded paths). */
  contentFields: FieldDescriptor[]
  /** The project-wide ignore list, shown with the content; absent when the engine does not describe it. */
  ignoreField?:  FieldDescriptor
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
  description: StorageFormDescription
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
  { type: 'invalid', errors: Record<string, string>, message?: string } |
  { type: 'saved' }
