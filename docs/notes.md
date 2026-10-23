# Notes: TODO, FIXME and other tagged comments

The **Notes** view (in the LoreMaster side bar, under Sync and Generators) lists the notes people leave in comments: `TODO`,
`FIXME`, `BUG`, `HACK`, `XXX`, `NOTE`, `WARN`. It needs nothing installed: the engine reads the files itself, so there is no
ripgrep to find and no language server to wait for.

```ts
// TODO: handle the empty list
let total = 0 // FIXME(alice): overflows past 2^53
```

```python
# BUG(#412): the second call returns stale data
```

```markdown
<!-- NOTE: this section is generated; edit the template -->
```

## What counts as a note

- A tag at the **start of a comment** (after the comment marker and any `*` down the side of a doc block): `// TODO: x`,
  `# FIXME x`, `/* HACK: x */`, `<!-- NOTE: x -->`. A tag in the middle of a sentence, or inside a string, is not a note.
- Brackets after the tag may hold an **owner**, an **issue** and a **date**, in any order: `TODO(alice)`, `TODO(#12)`,
  `TODO(2026-12-01)`, `TODO(alice, #12, 2026-12-01)`. A note whose date has passed is marked overdue.
- Comments are read the way each kind of file writes them: `//` and `/* */` (TypeScript, JavaScript, Go, Rust, Java, C#, Dart,
  Kotlin, Swift, C and C++, PHP, Vue and more), `#` (Python, shell, YAML, Ruby, TOML, Dockerfile and more, plus Python's
  docstrings), `--` (SQL, Lua, Haskell), `<!-- -->` (Markdown, HTML, XML, SVG), `;` and `%`. A file of a kind the engine does not know is
  not read.
- Files are read by the same rules a sync reads files by: `loreMaster.ignore`, `.gitignore` (unless `loreMaster.skipGitignored` is
  off), and never `node_modules`, `.git`, `dist` and the like. Files over 1 MB and binary files are skipped.

## The view

- Notes are grouped **by tag** (in the order of `loreMaster.notes.tags`) or **by file**; the button in the view title switches
  (`loreMaster.notes.groupBy`).
- A row shows the note's text, its owner, issue and date, and where it is; clicking it opens the file on the tag.
- The icon of the view shows how many notes there are. The view scans when it is shown and after files are saved, created, deleted
  or renamed; while it is hidden nothing is scanned.
- With several folders open, each is scanned and file groups are named after their folder.

## Settings

| Setting | |
|---|---|
| `loreMaster.notes.tags` | The tags to look for, upper case. Add your own (`WHY`, `DECISION`...); the order is the order of the groups. |
| `loreMaster.notes.groupBy` | `tag` (default) or `file`. |

The engine method behind it is `notes/scan` (`{ workspaceRoot, tags }`), which the other editors can use too.

## Not yet

Editor highlights and a gutter icon, a status bar count, filtering, Problems-panel diagnostics for overdue notes, a Markdown page of
the notes that syncs like any other, a command-line `notes` command, and turning a note into an issue are planned: see the issue
"gather TODO and other tagged comments as lore".
