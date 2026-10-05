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
