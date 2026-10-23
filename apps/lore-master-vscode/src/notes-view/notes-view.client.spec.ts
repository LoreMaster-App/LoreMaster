import { type NoteNode, type NotesScan } from './notes.contract'
import { NotesViewProvider } from './notes-view.client'

const scan: NotesScan = {
  notes: [
    { folder: '/w', path: 'src/a.ts', line: 3, column: 4, tag: 'TODO', text: 'first', owner: 'alice', issue: '12' },
    { folder: '/w', path: 'src/b.go', line: 9, column: 4, tag: 'FIXME', text: 'second', due: '2000-01-01' },
    { folder: '/w', path: 'src/a.ts', line: 20, column: 4, tag: 'TODO', text: '' },
  ],
  tags:         ['TODO', 'FIXME'],
  filesScanned: 7,
  problems:     [],
}

function provider (answer: Promise<NotesScan> | NotesScan, groupBy: 'tag' | 'file' = 'tag', found?: (scan: NotesScan) => void): { view: NotesViewProvider, asked: () => number } {
  let asked = 0
  const view = new NotesViewProvider({
    scan: () => {
      asked++

      return Promise.resolve(answer)
    },
    groupBy: () => groupBy,
  }, found)

  return { view, asked: () => asked }
}

const labels = (view: NotesViewProvider, rows: NoteNode[]): string[] => rows.map(row => String(view.getTreeItem(row).label))

describe('NotesViewProvider', () => {
  it('shows a group per tag with its count, and under it each note with where it is', async () => {
    const { view } = provider(scan)
    const groups = await view.getChildren()

    expect(labels(view, groups)).toEqual(['TODO', 'FIXME'])
    expect(groups.map(group => view.getTreeItem(group).description)).toEqual(['2', '1'])
    const notes = await view.getChildren(groups[0])
    expect(labels(view, notes)).toEqual(['first', '(TODO)'])
    expect(view.getTreeItem(notes[0]).description).toBe('alice · #12 · src/a.ts:3')
  })

  it('opens the file at the line and column of the tag', async () => {
    const { view } = provider(scan)
    const [todo] = await view.getChildren()
    const [first] = await view.getChildren(todo)
    const command = view.getTreeItem(first).command

    expect(command?.command).toBe('vscode.open')
    expect(command?.arguments?.[1]).toMatchObject({ selection: { startLine: 2, startCharacter: 3 } })
  })

  it('groups by file when asked, and says a note whose date has passed is overdue', async () => {
    const { view } = provider(scan, 'file')
    const files = await view.getChildren()

    expect(labels(view, files)).toEqual(['src/a.ts', 'src/b.go'])
    const [late] = await view.getChildren(files[1])
    expect(view.getTreeItem(late).description).toBe('overdue · due 2000-01-01 · FIXME · line 9')
  })

  it('scans once until it is told to scan again, and tells who is waiting what it found', async () => {
    let counted = 0
    const { view, asked } = provider(scan, 'tag', found => { counted = found.notes.length })

    await view.getChildren()
    await view.getChildren()
    expect(asked()).toBe(1)
    expect(counted).toBe(3)
    view.refresh()
    await view.getChildren()
    expect(asked()).toBe(2)
  })

  it('says so when there is nothing, and shows what could not be scanned', async () => {
    const { view } = provider({ notes: [], tags: ['TODO'], filesScanned: 4, problems: ['one: broke'] })
    const rows = await view.getChildren()

    expect(labels(view, rows)).toEqual(['one: broke', 'No notes found (4 files scanned for TODO).'])
  })

  it('shows the error of a scan that failed instead of throwing, and tries again next time', async () => {
    let calls = 0
    const view = new NotesViewProvider({
      scan: () => {
        calls++

        return calls === 1 ? Promise.reject(new Error('engine gone')) : Promise.resolve(scan)
      },
      groupBy: () => 'tag',
    })

    expect(labels(view, await view.getChildren())[0]).toContain('engine gone')
    expect(labels(view, await view.getChildren())).toEqual(['TODO', 'FIXME'])
  })
})
