# Getting started: syncing a workspace to Confluence

A step-by-step first run of the **LoreMaster** VS Code extension. It is a preview; the
first feature is syncing a workspace's Markdown into Confluence.

## 1. Install

In VS Code: **Extensions** → search **LoreMaster** (`LoreMaster.loremaster`) → **Install**.
It needs VS Code **1.96 or later**. The extension bundles its own engine, so there is
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
**`.lore-master.yaml`** in the workspace, so later syncs do not ask again.

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
- With several storages, each gets its own group.
- The tree follows `roots`, `excludes`, `ignore` and `skipGitignored` from
  `.lore-master.yaml`, so what you see is what a sync would read.

## 6. Change a storage's settings

In the **Storages** view, click the pencil on a storage (or run **LoreMaster: Edit storage**).
Pick a setting, then its new value:

- **Confluence:** title prefix, direction (`to-platform` or `two-way`), Mermaid diagrams
  (`image` or `code`), how pages link to each other, what to do when a title already exists,
  the folders to sync, and paths to leave out.
- **GitHub Pages:** the repository, the branch, the folders to sync, and paths to leave out.

The value is checked and saved to `.lore-master.yaml` with your comments kept; a value that is
not allowed is refused with the reason and nothing is written. For anything else, **Open
config** opens the file.

## What you get

- A page tree mirroring your files: nesting from `parent:` annotations, dotted filenames
  (`readme.architecture.md` under `readme.md`), directory `README.md` indexes, then the
  parent you chose.
- Each synced file gains a small `<!-- lore-master … -->` annotation recording its page, so
  a re-run changes only what actually changed — it never makes duplicates.

## Configuration

`.lore-master.yaml` is created for you; see the schema and options (space, `titlePrefix`,
`mermaidMode`, `titleCollision`, `linkMode`, content roots) in the project
[`README.md`](../README.md).

## Troubleshooting

- **"not compatible with this version of Visual Studio Code"** — update VS Code to 1.96+.
- **No connection yet** — run *LoreMaster: Add Connection* first; a sync with no connection
  just prompts you to add one.
- **A custom engine build** — point `loreMaster.engine.path` at a binary if you are
  developing the engine; otherwise the bundled one is used.
