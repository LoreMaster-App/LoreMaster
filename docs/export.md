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
| `loreMaster.export.docxTemplate` | A Word file used as the [company template](#company-templates) for DOCX exports. Relative to the workspace folder, or absolute. |
| `loreMaster.export.odtTemplate` | An OpenDocument file used as the company template for ODT exports. |
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
| `asciidoc` | AsciiDoc for Asciidoctor and Antora (`== Heading`, `**bold**`, `[source,go]`, `\|===` tables). Links to other Markdown files become `xref:name.adoc`. | `.adoc` |
| `rst` | reStructuredText for Sphinx (underlined headings, `**bold**`, `.. code-block::`, `.. list-table::`). Links to other Markdown files become `:doc:`; strike-through has no equivalent and is dropped. | `.rst` |
| `slack` | A Slack message (mrkdwn): `*bold*`, `_italic_`, `~strike~`, `<url\|label>`. Slack has no headings, tables or rules, so a heading is a bold line, a table a code block and a rule a line of dashes. | `.slack.txt` |
| `org` | Org-mode for Emacs (`* Heading`, `*bold*`, `/italic/`, `#+begin_src`, `\| tables \|`). | `.org` |
| `latex` | A LaTeX file that compiles on its own (`article` with hyperref, listings, graphicx), with `\section`, `itemize`, `tabular`. | `.tex` |
| `outline` | A JSON outline of the document: `title`, `headings` (level, text, anchor), `links` and `tasks` (text, done), for tooling. | `.outline.json` |
| `man` | A roff man page, section 7 (`.TH`, `.SH`, `.IP`, `tbl` tables). The first level-1 heading is the page title. | `.7` |
| `rtf` | Rich Text Format for Word, WordPad and other word processors (headings, lists, tables, hyperlinks). | `.rtf` |
| `mdx` | MDX for Docusaurus: Markdown with a `title` front matter block, `{ }` and `<` escaped, raw HTML left out. | `.mdx` |

Front matter and the sync annotation are never part of an export.

## Company templates

A Word or OpenDocument export can take the look of your company's template: its styles, headers and footers, page size and
margins, and its theme, with the template's own text left out. Point `loreMaster.export.docxTemplate` or
`loreMaster.export.odtTemplate` at the file, or on the command line:

```bash
lore-master-engine export --format docx --reference templates/acme.docx docs/guide.md
lore-master-engine export --format odt  --reference templates/acme.odt  docs/guide.md
```

How the template and the document meet:

- **Styles are matched by name.** A heading takes the template's "Heading 1" whatever the template calls it internally or
  which language it is in; the same goes for Heading 2 to 6, Quote, List Paragraph, Hyperlink and Table Grid. A style the
  template does not have (the code styles, usually) is added to it, based on the template's own default text style, so the
  document still reads as part of the template.
- **Headers, footers and page setup come from the template**: logo, page numbers, a confidential line, A4 or Letter, margins.
  In an OpenDocument template, the pictures its header uses travel with it.
- **A Word template's numbering is kept**, so a template whose headings are numbered still numbers them; the document's lists
  are added beside it and never take the template's list ids.
- The template's own text and tables are not used. Make the template a file with the styles and the page the way you want
  them, plus whatever sits in the header and footer.
- Only the format the template is for takes it: a `.docx` is the template of `docx`, an `.odt` of `odt`. A file that is not
  the right kind is refused with the reason.

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
- **PDF and PNG** have no page headers or footers. From the editor they draw Mermaid diagrams (with the Mermaid library the
  extension ships); a diagram with an error stays as code. On the command line, and from the JetBrains plugin, the engine uses the
  copy built into it (written once to the temporary folder); `--mermaid PATH` or `LORE_MASTER_MERMAID` (a file) name another.
- **Mermaid** diagrams are drawn in PDF, PNG and HTML when there is a Mermaid library and a browser (see above for PDF and
  PNG). The HTML export then holds each diagram as an inline SVG, so the file still needs no script and can be opened
  anywhere; a diagram that cannot be drawn, or an export with no library or browser, keeps the diagram as a code block
  without an error. Plain text, Word, OpenDocument and the wiki formats keep them as code.
- **Links to other Markdown files** become `[[Page]]` links in MediaWiki and `.html` links in HTML. Elsewhere
  they keep the path.

## Still to come

The GitHub wiki format, page headers and footers, and more formats
are tracked in the epic [#405](https://github.com/LoreMaster-App/LoreMaster/issues/405).
