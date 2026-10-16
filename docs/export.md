# Exporting Markdown

LoreMaster converts a Markdown file to other formats without any storage, settings or credentials.
It is the same converter whether you call it from the command line, a CI job or (soon) the editor.

```bash
lore-master-engine export --list
lore-master-engine export --format jira-wiki docs/guide.md          # writes docs/guide.jira.txt
lore-master-engine export --format html --out dist docs/*.md        # one file per document
lore-master-engine export --format mediawiki --stdout docs/guide.md # print it
```

## Formats

| Id | Result | File ending |
|---|---|---|
| `html` | One self-contained page: styles inlined, local images embedded, light and dark, print-ready. Links to other `.md` files point at the matching `.html`. | `.html` |
| `txt` | The text without Markdown syntax: headings underlined, lists kept, tables aligned, links as `text (url)`. | `.txt` |
| `jira-wiki` | Jira wiki markup (`h1.`, `*bold*`, `{code:go}`, `\|\|table\|\|`). | `.jira.txt` |
| `jira-adf` | Atlassian Document Format JSON for Jira Cloud and Confluence Cloud, with real task lists and tables. | `.adf.json` |
| `mediawiki` | Wikipedia / MediaWiki wikitext (`== Heading ==`, `'''bold'''`, `{\| class="wikitable"`, `<syntaxhighlight>`). | `.wiki` |

Front matter and the sync annotation are never part of an export.

## What each format does with what it cannot say

- **Images** in `jira-adf` become a link, because ADF images need an uploaded media id. In `jira-wiki` and
  `mediawiki` they point at the file by name; attach it to the issue or upload it to the wiki.
- **Nested structures** that ADF does not allow inside a list item (a table, a quote) are written as plain
  paragraphs of their text, so the document stays valid.
- **Mermaid** blocks stay code blocks in every format for now.
- **Links to other Markdown files** become `[[Page]]` links in MediaWiki and `.html` links in HTML. Elsewhere
  they keep the path.

## Still to come

PDF, PNG, Word and OpenDocument, the GitHub wiki format, the editor command and settings, and more formats
are tracked in the epic [#405](https://github.com/LoreMaster-App/LoreMaster/issues/405).
