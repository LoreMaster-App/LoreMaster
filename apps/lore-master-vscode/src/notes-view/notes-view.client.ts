import { join } from 'node:path'
import * as vscode from 'vscode'
import { groupNotes, isOverdue } from './group-notes.algorithm'
import { styleOfTag } from './note-tag-style.config'
import type { FoundNote, NoteGroup, NoteNode, NotesGroupBy, NotesScan } from './notes.contract'

/** The view id contributed in package.json. */
export const NOTES_VIEW_ID = 'loreMaster.notes'

/** What the view needs from the rest of the extension. */
export interface NotesViewSource {
  /** Scans every workspace folder for notes. */
  scan (): Promise<NotesScan>
  groupBy (): NotesGroupBy
}

const today = (): string => new Date().toISOString().slice(0, 10)

/** What follows a note's text in its row: its owner, its issue and its date. */
function details (note: FoundNote): string {
  const parts = [note.owner, note.issue === undefined || note.issue === '' ? undefined : `#${note.issue}`, note.due === undefined || note.due === '' ? undefined : `due ${note.due}`]

  return parts.filter(part => part !== undefined && part !== '').join(' · ')
}

/**
 * The Notes view: the tagged comments of the workspace (TODO, FIXME...) grouped by tag or by file, each opening its line. The
 * scan is the engine's, so nothing has to be installed; the view asks for one when it first opens and whenever it is told files
 * have changed.
 */
export class NotesViewProvider implements vscode.TreeDataProvider<NoteNode> {
  private readonly changed = new vscode.EventEmitter<NoteNode | undefined>()
  private scanning: Promise<NotesScan> | undefined
  private last:     NotesScan | undefined

  readonly onDidChangeTreeData = this.changed.event

  constructor (private readonly source: NotesViewSource, private readonly found: (scan: NotesScan) => void = () => {}) {}

  private async load (): Promise<NotesScan> {
    const scan = await this.source.scan()
    this.last = scan
    this.found(scan)

    return scan
  }

  private scan (): Promise<NotesScan> {
    this.scanning ??= this.load()

    return this.scanning
  }

  private groupItem (group: NoteGroup): vscode.TreeItem {
    const item = new vscode.TreeItem(group.label, vscode.TreeItemCollapsibleState.Expanded)
    item.id = group.id
    item.description = String(group.notes.length)
    item.iconPath = group.tag === undefined
      ? vscode.ThemeIcon.File
      : new vscode.ThemeIcon(styleOfTag(group.tag).icon, new vscode.ThemeColor(styleOfTag(group.tag).color))
    if (group.tag === undefined) { item.resourceUri = vscode.Uri.file(join(group.notes[0].folder, group.notes[0].path)) }

    return item
  }

  private noteItem (note: FoundNote, groupedBy: NotesGroupBy): vscode.TreeItem {
    const item = new vscode.TreeItem(note.text === '' ? `(${note.tag})` : note.text, vscode.TreeItemCollapsibleState.None)
    const overdue = isOverdue(note, today())
    const style = styleOfTag(note.tag)
    const place = groupedBy === 'tag' ? `${note.path}:${note.line}` : `${note.tag} · line ${note.line}`
    item.description = [overdue ? 'overdue' : undefined, details(note), place].filter(part => part !== undefined && part !== '').join(' · ')
    item.iconPath = new vscode.ThemeIcon(style.icon, new vscode.ThemeColor(overdue ? 'errorForeground' : style.color))
    item.tooltip = [`${note.tag}${details(note) === '' ? '' : ` (${details(note)})`}: ${note.text}`, `${note.path}:${note.line}`].join('\n')
    item.command = {
      command:   'vscode.open',
      title:     'Open',
      arguments: [
        vscode.Uri.file(join(note.folder, note.path)),
        { selection: new vscode.Range(note.line - 1, Math.max(note.column - 1, 0), note.line - 1, Math.max(note.column - 1, 0)) },
      ],
    }

    return item
  }

  /** Scans again and redraws. */
  refresh (): void {
    this.scanning = undefined
    this.changed.fire(undefined)
  }

  /** Redraws what was scanned already, grouped as the settings say now. */
  redraw (): void {
    this.changed.fire(undefined)
  }

  async getChildren (node?: NoteNode): Promise<NoteNode[]> {
    if (node?.kind === 'group') {
      return node.group.notes.map(note => ({ kind: 'note', note, groupedBy: this.source.groupBy() }))
    }
    if (node) { return [] }

    let scan: NotesScan
    try {
      scan = await this.scan()
    } catch (error) {
      this.scanning = undefined

      return [{ kind: 'message', text: `The notes could not be scanned: ${error instanceof Error ? error.message : String(error)}` }]
    }
    const rows: NoteNode[] = scan.problems.map(text => ({ kind: 'message', text }))
    if (scan.notes.length === 0) {
      rows.push({ kind: 'message', text: `No notes found (${scan.filesScanned} files scanned for ${scan.tags.join(', ')}).` })

      return rows
    }

    return [...rows, ...groupNotes(scan.notes, scan.tags, this.source.groupBy()).map((group): NoteNode => ({ kind: 'group', group }))]
  }

  getTreeItem (node: NoteNode): vscode.TreeItem {
    switch (node.kind) {
      case 'group': { return this.groupItem(node.group)
      }
      case 'note': { return this.noteItem(node.note, node.groupedBy)
      }
      case 'message': { return new vscode.TreeItem(node.text, vscode.TreeItemCollapsibleState.None)
      }
    }
  }

  /** The notes of the last scan, for a count. */
  get count (): number {
    return this.last?.notes.length ?? 0
  }
}
