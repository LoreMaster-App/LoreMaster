# Getting started: syncing a workspace to Confluence

A step-by-step first run of the **LoreMaster** VS Code extension. It is a preview. This page
syncs a workspace's Markdown into Confluence; GitHub Pages and the wiki are set up the same way
(**Add sync storage**), and the [next steps](#next-steps) add generated pages and CI.

## 1. Install

In VS Code: **Extensions** → search **LoreMaster** (`LoreMaster.loremaster`) → **Install**.
It needs VS Code **1.101 or later**. The extension bundles its own engine, so there is
nothing else to install.

## 2. Open a workspace

Open the folder whose Markdown you want to sync. Each `.md` file becomes a page; the first
`# H1` is its title.

## 3. Add a connection

Command Palette (`Ctrl`/`Cmd`+`Shift`+`P`) → **LoreMaster: Add Connection** — or the
**Accounts menu** (bottom-left) → *Sign in with Confluence*. Then:

1. Enter the site URL (any page URL of the site works), e.g.
   `https://your-site.atlassian.net/wiki` (Cloud) or `https://confluence.example.com`
   (Data Center). The edition is detected for you.
2. Choose how to sign in:
   - **Cloud** — your Atlassian account email + an
     [API token](https://id.atlassian.com/manage-profile/security/api-tokens).
   - **Data Center** — a personal access token.
3. The connection is verified and saved. The secret lives in VS Code's secret store, never
   in your repository.

## 4. Sync

Command Palette → **LoreMaster: Sync** (or **Sync Current File** for just the
open file).

The **first** sync asks where things go — the space, the parent page everything nests
under, and a title prefix (pages are titled `<prefix>: <H1>`). Your answers are written to
**`.vscode/settings.json`** of the workspace (`loreMaster.outputs`), so later syncs do not ask again.

Then LoreMaster shows a **plan** — what it would create, update, move, or leave unchanged —
and waits for you to confirm before writing anything. Confirm, and it creates the pages
parents-first, uploads images, renders Mermaid diagrams to images, and links pages to each
other by title.

## 5. See where everything stands: the Pages view

Open the LoreMaster icon in the activity bar and expand **Pages**. It lists every Markdown
file as the tree it is, or will be, on the storage — the same nesting the sync applies —
each with an icon:

| Icon | Meaning |
|---|---|
| `+` new | Not on the platform yet; the next sync creates it. |
| ✓ synced | The file is the one last synced. |
| ✎ local changes | The file changed since the last sync; the next sync updates the page. |
| ☁↓ remote changes | The page was edited on the platform since the last sync. |
| ⚠ conflict | Both the file and the page changed. |
| 🔗 will adopt | A page with this title exists; the next sync takes it over. |

Without a connection the view knows the **local** half: new, synced or local changes, read
from the files and their annotations. Click **Check remote status** (the cloud button in the
view's title bar) to ask the platform too — it only reads, and adds remote changes,
conflicts and pages whose file is gone. The answer is dropped again as soon as a file
changes, so the view never shows a platform state it has not just asked for.

- **Titles or file names:** the button next to it switches how pages are named — their
  content title (first heading) or their file name. The other one is shown beside it, and
  the choice is the `loreMaster.pages.label` setting.
- Click a page to open its file; the cloud-upload button on a row syncs just that page.
- A **Local** node above the storages lists every Markdown file in the repository and which
  storages sync it; each storage gets its own group.
- The tree follows `roots`, `excludes`, `ignore` and `skipGitignored` from
  your LoreMaster settings, so what you see is what a sync would read.
- **Left out** (collapsed, under the pages) lists the Markdown files the sync does not read, each
  with the setting that left it out: the `ignore` list, the storage's `excludes`, a `.gitignore`,
  or being outside the folders to sync. Click one to open it. `lore-master-engine tree` prints
  the same list.

### Leaving files out

A file is left out if any of these applies, unless a storage's `include:` takes it back with a
more specific entry:

| Setting | Applies to | |
|---|---|---|
| `ignore:` (top level) | every storage | gitignore-style patterns, e.g. `CLAUDE.md`, `internal/` |
| `exclude:` / `excludes:` (on a storage / its content entry) | that storage | same patterns |
| `include:` (on a storage) | that storage | takes a file or folder back from an exclusion |
| `skipGitignored` | the workspace | `true` by default: Markdown a `.gitignore` ignores is not synced |

Change the `ignore` list from **Edit storage → Ignored paths (all storages)**, or in the file,
and right-click a page or folder in the **Pages** view for **Do not sync…**. The **Local** node
lists every file with where it goes. See [Choosing what goes where](choosing-what-goes-where.md).

## 6. Change a storage's settings

In the **Storages** view, click the pencil on a storage (or run **LoreMaster: Edit storage**). A form
opens with every setting of the storage at once:

- **Confluence:** the connection, the **space** (browse the space list or type its key), the **parent
  page** (search for it by title; the form fills in its id), the title prefix, direction (`to-platform`
  or `two-way`), Mermaid diagrams (`image` or `code`), how pages link to each other, what to do
  when a title already exists, the folders to sync, and paths to leave out.
- **GitHub Pages / Wiki:** the repository, the branch, the folder inside the branch (so another site
  can share it), the folders to sync, and paths to leave out.

**Add sync storage** opens the same form, empty: nothing is preselected, so you choose where to sync,
fill the fields and save once.

Saving checks the values and writes them to the workspace's `.vscode/settings.json` with your
comments kept; a value that is not allowed is refused with the reason and nothing is written.

### What changing the parent page or the space does

The parent page is read at every sync, not stored in the files: a page's annotation only remembers
which site and space it was synced to. So when you change the **parent page** of a storage, the next
sync plans a **move** for every page that sits directly under the old parent, and the pages below
them follow. Nothing needs deleting first. Changing the **space** (or the site) is different: files
annotated for another space are treated as new, so the pages are created in the new space and the
old ones stay where they were. To undo a sync that went to the wrong place, see
[Undo a sync](#undo-a-sync).

## Undo a sync

Every Confluence sync is journalled on your machine: the pages it created, the pages it updated
(with how they were before) and the local files it rewrote (with their earlier content).
**LoreMaster: Roll back a sync** (or **Roll back** on the notification after a sync) shows what it
would do and asks before doing it:

- pages the sync **created** are deleted, children first;
- pages it **updated or moved** are put back as they were (title, parent and body);
- local files get their earlier content back, so the annotations are reverted too.

Anything edited after the sync, on the platform or in the file, is left alone and reported; a second
prompt offers to discard those edits as well. Pages a sync moved to the trash are reported, since the
platform's own trash restores them. The command line has the same: `lore-master-engine rollback`
(see [the command line](cli.md#undo-a-sync)).

## What you get

- A page tree mirroring your files: nesting from `parent:` annotations, dotted filenames
  (`readme.architecture.md` under `readme.md`), directory `README.md` indexes, then the
  parent you chose.
- Each synced file gains a small `<!-- lore-master … -->` annotation recording its page, so
  a re-run changes only what actually changed — it never makes duplicates.

## Next steps

- **Pages written from your reports and code.** Open the **Generators** view, **Add generator**, pick
  test results, coverage, OpenAPI, Storybook or a code documenter, then run it. The pages appear in
  your workspace and the next sync publishes them. See [generators](generators.md).
- **Keep it fresh while you type.** Run **LoreMaster: Toggle watch mode**; see [watch mode](watch.md).
- **Run it on CI.** Run **LoreMaster: Add to your CI pipeline**: it reads your settings, finds
  whether the repository uses GitHub Actions or Azure Pipelines (and asks when it cannot tell) and
  opens a guide with the pipeline and the secrets to create; see [CI session](cli.md#the-ci-session).
  The engine is also a command line with a [GitHub Action](github-action.md); see
  [the command line](cli.md).
- **Let an AI agent write the docs your way.** See [the documentation agent](agent.md) and
  [the MCP server](mcp-server.md).

## Configuration

The settings are `loreMaster.*` in the editor's settings; see [configuration](configuration.md) for the
layers (folder > workspace > user) and the options (space, `titlePrefix`, `mermaidMode`,
`titleCollision`, `linkMode`, content roots), which the Settings UI also describes.

## Troubleshooting

- **"not compatible with this version of Visual Studio Code"** — update VS Code to 1.101+.
- **No connection yet** — run *LoreMaster: Add Connection* first; a sync with no connection
  just prompts you to add one.
- **A custom engine build** — point `loreMaster.engine.path` at a binary if you are
  developing the engine; otherwise the bundled one is used.
