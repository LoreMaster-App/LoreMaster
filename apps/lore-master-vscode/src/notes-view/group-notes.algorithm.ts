import type { FoundNote, NoteGroup, NotesGroupBy } from './notes.contract'

const byPlace = (a: FoundNote, b: FoundNote): number =>
  a.folder.localeCompare(b.folder) || a.path.localeCompare(b.path) || a.line - b.line

/** The name of a file in a group by file: its path, with the folder's name in front when there is more than one folder. */
function fileLabel (note: FoundNote, severalFolders: boolean): string {
  if (!severalFolders) { return note.path }
  const name = note.folder.replace(/[/\\]+$/, '').split(/[/\\]/).at(-1) ?? note.folder

  return `${name}/${note.path}`
}

/**
 * The notes in groups: by tag (in the order of `tags`, a tag with no note left out, a tag that is not in the list last) or by file
 * (in path order). Inside a group the notes are in file then line order.
 */
export function groupNotes (notes: FoundNote[], tags: string[], groupBy: NotesGroupBy): NoteGroup[] {
  const sorted = [...notes].sort(byPlace)
  if (groupBy === 'tag') {
    const order = [...tags, ...new Set(sorted.map(note => note.tag).filter(tag => !tags.includes(tag)))]

    return order
      .map(tag => ({ id: `tag:${tag}`, label: tag, tag, notes: sorted.filter(note => note.tag === tag) }))
      .filter(group => group.notes.length > 0)
  }

  const severalFolders = new Set(sorted.map(note => note.folder)).size > 1
  const groups: NoteGroup[] = []
  const files = new Map<string, NoteGroup>()
  for (const note of sorted) {
    const key = `${note.folder}\u{0}${note.path}`
    let group = files.get(key)
    if (group === undefined) {
      group = { id: `file:${key}`, label: fileLabel(note, severalFolders), notes: [] }
      files.set(key, group)
      groups.push(group)
    }
    group.notes.push(note)
  }

  return groups
}

/** Whether a note's date has passed, today being `today` (YYYY-MM-DD). A note with no date is never overdue. */
export function isOverdue (note: FoundNote, today: string): boolean {
  return note.due !== undefined && note.due !== '' && note.due < today
}
