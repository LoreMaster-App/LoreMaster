# Editing Markdown

LoreMaster edits Markdown itself, so a separate Markdown extension is not needed for the everyday work: formatting
shortcuts, lists that continue and renumber, and tables that stay lined up. All of it works in any Markdown file, on
every cursor at once, and each part has a setting to turn it off. (Exporting is in [Exporting Markdown](export.md).)

VS Code's own Markdown support already completes links to files and headings, pastes a URL over selected text as a link
and pastes images, so LoreMaster leaves those to it.

## Formatting shortcuts

Each is also a command in the palette (**LoreMaster Markdown: ...**) and can be rebound. Turn the keys off with
`loreMaster.markdown.shortcuts`.

| Keys (Mac: `Cmd` for `Ctrl`) | Does |
|---|---|
| `Ctrl+B` | Bold |
| `Ctrl+I` | Italic |
| `Alt+S` | Strikethrough |
| `Alt+C` | Inline code |
| `Alt+Shift+C` | Code block |
| `Alt+Q` | Block quote |
| `Alt+T` | Task: a plain line or list item becomes an unchecked task; pressing again ticks it, and again unticks it |
| `Ctrl+Shift+]` / `Ctrl+Shift+[` | Heading one level deeper / shallower; a plain line becomes a heading, and a first-level heading becomes text again |

Pressing a shortcut again takes the formatting off. With nothing selected it formats the word at the cursor, or, away from a
word, leaves an empty pair of markers to type between. Different kinds never disturb each other: italic on bold text
gives `***text***`, and turning italic off leaves `**text**`. The code block shortcut fences the selected lines, or on an
empty line opens an empty block with the cursor in it, and on a fenced block it takes the fences off.

## Lists

With `loreMaster.markdown.listEditing` on (the default), on a list item:

- **Enter** starts the next item: the next number for a numbered list, an empty box for a task, the same marker for a bullet,
  and what was after the cursor moves down into it. The items below are renumbered. On an empty item it ends the list, and a
  nested empty item comes out a level instead.
- **Tab** nests the item (by the width of its marker, so two spaces under `-` and three under `1.`) and **Shift+Tab** brings it
  out. Numbered lists are renumbered: the first item of a nested list starts at 1, and an item that comes out follows the one
  before it. An item with nothing above it cannot be nested, and then Tab does what it always does.
- A list keeps the number it starts at, and a list written with every number the same (`1.` throughout) stays that way.

Enter inside the marker, and Tab or Enter anywhere else, are not touched. Block quotes around a list are kept.

## Tables

With `loreMaster.markdown.tableEditing` on (the default), inside a table:

- **Tab** and **Shift+Tab** move between cells and line the whole table up as they go: pipes aligned, every cell padded, the
  delimiter row as wide as its column, and `:--`, `:-:` and `--:` alignments kept (the text is placed accordingly). Wide
  characters such as Chinese and emoji count as two columns. Tab in the last cell adds a row.
- **Format Table**, **Insert Row Above/Below**, **Delete Row**, **Insert Column Left/Right** and **Delete Column** are in the
  palette. The header row cannot be deleted, nor the last column.
- `loreMaster.markdown.formatTablesOnSave` (off by default) lines up every table in a file when it is saved, leaving tables
  inside fenced code alone.

A pipe written `\|` is text, as on GitHub; a pipe inside a code span still divides cells, as GitHub does.

## Settings

| Setting | Default | |
|---|---|---|
| `loreMaster.markdown.shortcuts` | on | The formatting shortcuts |
| `loreMaster.markdown.listEditing` | on | Enter, Tab and Shift+Tab in lists |
| `loreMaster.markdown.tableEditing` | on | Tab and Shift+Tab in tables |
| `loreMaster.markdown.formatTablesOnSave` | off | Line tables up on save |
