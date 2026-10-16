# Exporting Markdown

LoreMaster converts a Markdown file to other formats without any storage, settings or credentials.
It is the same converter whether you call it from the command line, a CI job or the editor.

```bash
lore-master-engine export --list
lore-master-engine export --format jira-wiki docs/guide.md          # writes docs/guide.jira.txt
lore-master-engine export --format html --out dist docs/*.md        # one file per document
lore-master-engine export --format mediawiki --stdout docs/guide.md # print it
```

## In the editor

- **LoreMaster: Export Markdown...** is on the editor title bar of a Markdown file, in the explorer's context menu
  for a file or a folder (every Markdown file in it, skipping `node_modules` and `.git`), and in the command
  palette. Select several files in the explorer to export them together. It asks which formats to produce, and you
  can tick more than one: each becomes its own file. Unsaved changes in the editor are exported, not the file on disk.
- **LoreMaster: Copy Markdown As...** is in the editor's context menu and the command palette. It puts the document,
  or just the selection when there is one, on the clipboard as Jira wiki markup, Jira Cloud ADF, MediaWiki, HTML or
  plain text, ready to paste. Word and OpenDocument are files, so they are not offered.

| Setting | Meaning |
|---|---|
| `loreMaster.export.formats` | The formats ticked by default (default `["html"]`). |
| `loreMaster.export.pageSize` | The paper of a PDF: `A4`, `A3`, `A5`, `Letter` or `Legal`. |
| `loreMaster.export.pngWidth` | The width of a PNG in pixels. |
| `loreMaster.export.browserPath` | The Chrome, Edge or Chromium to print with, when the one installed in the usual place is not the one you want. |
| `loreMaster.export.folder` | Where exports go. Blank writes each next to its Markdown file. A folder, relative to the workspace or absolute, receives them with the document's own folders mirrored below it, so `docs/guide.md` exported to `dist` becomes `dist/docs/guide.html`. |

Behind both commands are the engine's `export/formats` and `export/run` calls, which any editor shell can use.

## Formats

| Id | Result | File ending |
|---|---|---|
| `html` | One self-contained page: styles inlined, local images embedded, light and dark, print-ready. Links to other `.md` files point at the matching `.html`. | `.html` |
| `docx` | A Word file: real Heading styles (so Word's navigation pane and table of contents work), numbered and bulleted lists that restart where the Markdown does, tables with a repeating header row, code in a shaded monospace style, quotes, task boxes, hyperlinks and embedded PNG, JPEG and GIF images. | `.docx` |
| `odt` | The same as an OpenDocument text file, for LibreOffice and anywhere an open standard is required. | `.odt` |
| `pdf` | The HTML export printed to PDF on the paper you choose (A4 by default). Always light, whatever your system's theme. Needs Chrome, Edge or Chromium installed. | `.pdf` |
| `png` | One image of the whole page, as wide as you choose (1000 px by default) and as tall as the page. Always light. Needs Chrome, Edge or Chromium installed. | `.png` |
| `txt` | The text without Markdown syntax: headings underlined, lists kept, tables aligned, links as `text (url)`. | `.txt` |
| `jira-wiki` | Jira wiki markup (`h1.`, `*bold*`, `{code:go}`, `\|\|table\|\|`). | `.jira.txt` |
| `jira-adf` | Atlassian Document Format JSON for Jira Cloud and Confluence Cloud, with real task lists and tables. | `.adf.json` |
| `mediawiki` | Wikipedia / MediaWiki wikitext (`== Heading ==`, `'''bold'''`, `{\| class="wikitable"`, `<syntaxhighlight>`). | `.wiki` |

Front matter and the sync annotation are never part of an export.

## PDF and PNG need a browser

LoreMaster does not ship a browser. For `pdf` and `png` it runs the one already on the machine, headless and with a
profile of its own, so a Chrome you have open is not disturbed. It looks for Google Chrome, then Microsoft Edge (every
Windows 10 and 11 has it), then Chromium, in the usual places, and for the usual command names on Linux. To use another,
set `loreMaster.export.browserPath`, pass `--browser PATH` on the command line, or set `LORE_MASTER_BROWSER`. A path that
does not exist is an error and not a reason to quietly use a different browser. With none installed the export says so
and says how to point at one. On a CI runner, GitHub's hosted images already have Chrome.

```bash
lore-master-engine export --format pdf --page-size Letter docs/guide.md
lore-master-engine export --format png --width 1400 --browser /usr/bin/chromium docs/guide.md
```

## What each format does with what it cannot say

- **Images** in `jira-adf` become a link, because ADF images need an uploaded media id. In `jira-wiki` and
  `mediawiki` they point at the file by name; attach it to the issue or upload it to the wiki.
- **Nested structures** that ADF does not allow inside a list item (a table, a quote) are written as plain
  paragraphs of their text, so the document stays valid.
- **Word and OpenDocument** embed PNG, JPEG and GIF images that sit beside the file; SVG, web images and missing
  files are written as their alt text. Links to other Markdown files point at the matching `.docx` or `.odt`.
- **PDF and PNG** have no page headers or footers, and Mermaid blocks are printed as code.
- **Mermaid** blocks stay code blocks in every format for now.
- **Links to other Markdown files** become `[[Page]]` links in MediaWiki and `.html` links in HTML. Elsewhere
  they keep the path.

## Still to come

Applying a company Word template, the GitHub wiki format, Mermaid drawn in HTML, PDF and PNG, page headers and footers, and more formats
are tracked in the epic [#405](https://github.com/LoreMaster-App/LoreMaster/issues/405).
