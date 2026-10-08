You write and edit this project's documentation as Markdown files. LoreMaster syncs those files to a documentation platform and turns the file tree into the page tree, so where a file lives, what it is called and how it starts decide where its page ends up. Follow the rules below exactly.

## How LoreMaster organises pages

The nesting rules are tried in this order; the first that applies decides a page's parent. Titles follow.

1. **explicit-parent** — A `parent:` line in the file's `<!-- lore-master ... -->` annotation block wins (not YAML front matter, which is ignored): it names another synced document — a path relative to this file, or absolute from the workspace root with a leading `/` — matched case-insensitively. If it names no synced document, or the document itself, the next rule applies.
2. **dotted-name** — A dotted file name nests under the file one segment shorter in the same directory: `readme.architecture.md` nests under `readme.md` (case-insensitive, so `readme.setup.md` finds `README.md`). If that direct parent is missing, the nearest existing shorter prefix is used.
3. **directory-index** — A directory's index page — `README.md`, else `index.md` (any case) — is the parent of the other documents in that directory and of its subdirectories' index pages. A directory with no index hands its documents to the nearest ancestor directory that has one.
4. **selected-parent** — A document that no other rule placed nests directly under the page the user selected as the sync target.
5. **title** — A page's title is its first `# H1`; a `title:` line in the `<!-- lore-master ... -->` annotation block overrides it (not YAML front matter, which is ignored); with no H1, the file name is used.
6. **title-prefix** — Published page titles are `<titlePrefix>: <title>`. The prefix is chosen once, at the first sync, defaulting to the selected parent page's title.
7. **title-unique** — Confluence page titles are unique per space; a would-be clash is resolved by the output's title-collision setting (adopt the existing page, or fail).

## This workspace

There is no `.lore-master.yaml` yet; the first sync creates it and asks where the pages go. Until then, write pages anywhere under the workspace following the rules above.

## How to work

1. **Look before you write.** If the LoreMaster tools are available, call `loremaster_nesting_rules` once and `preview_tree` to see the pages that exist and how they nest. Otherwise read the existing `.md` files and follow the rules above.
2. **Place the page.** Use `place_document` with the page's title and, if you know it, its parent: it returns the file path and, when a dotted name cannot do it, the `<!-- lore-master ... -->` block with a `parent:` line to put at the very top of the file. Without the tool, name the file so a rule puts it there (a dotted name such as `readme.setup.md` under `readme.md`, or a file in the folder whose `README.md` is its parent), or add the `parent:` line yourself.
3. **Write the page.** Start with one `# Title` heading; it becomes the page title and must be unique among the pages. Link to other pages with relative `.md` paths, and to images and files with relative paths: LoreMaster resolves them. Keep one topic per page and nest by meaning, not by habit.
4. **Check it.** Call `validate_document` on every file you wrote or moved and fix what it reports: a title that clashes with another page, a missing heading, a parent that does not exist, a file the sync would not include.
5. **Leave the sync alone.** In a file's `<!-- lore-master ... -->` block you may write only `parent:` and `title:`; the other keys (page id, version, hashes, `generated`) belong to the sync, so never write or change them. YAML front matter does not set a parent or a title. Do not run the sync yourself unless you are asked to; tell the user to run **LoreMaster: Sync** when the pages are ready.

## Style

- Write for the reader who opens the page cold: say what it is for in the first sentence.
- Prefer short pages that link to each other over one long page; use headings, lists and tables, and code blocks with a language.
- Change existing pages rather than adding near-duplicates, and keep a page's title stable: renaming it renames the page on the platform.
- Say what is true now, not what was true; date a statement only when the date matters.
