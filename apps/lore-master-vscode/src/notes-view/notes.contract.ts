import type { NoteEntry } from '../engine-protocol'

/** How the notes are grouped in the view. */
export type NotesGroupBy = 'tag' | 'file'

/** A note and the workspace folder it was found in. */
export interface FoundNote extends NoteEntry {
  /** The workspace folder, as an absolute path. */
  folder: string
}

/** What a scan of every workspace folder found. */
export interface NotesScan {
  notes:        FoundNote[]
  /** The tags that were looked for, in the order the settings give them. */
  tags:         string[]
  filesScanned: number
  /** Why a folder could not be scanned, if any. */
  problems:     string[]
}

/** A group of notes: all of one tag, or all of one file. */
export interface NoteGroup {
  id:    string
  label: string
  /** The tag of a group by tag; absent for a group by file. */
  tag?:  string
  notes: FoundNote[]
}

/** A row of the Notes view. */
export type NoteNode =
  | { kind: 'group', group: NoteGroup } |
  { kind: 'note', note: FoundNote, groupedBy: NotesGroupBy } |
  { kind: 'message', text: string }
