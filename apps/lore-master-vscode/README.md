# LoreMaster

![LoreMaster](https://raw.githubusercontent.com/LoreMaster-App/LoreMaster/main/assets/lore-master-concept-md.jpg)

**LoreMaster gathers a project's lore and publishes it where your team already reads.**
Keep your docs as plain Markdown in your repo, and LoreMaster syncs them to **Confluence**
or publishes them as a **static site on GitHub Pages** — directly from VS Code.

> **Preview.** The engine and its libraries are built and tested; this extension is
> published as a preview while the end-to-end sync is hardened.

## Features

- **Sync Markdown → Confluence.** Every `.md` file in your workspace becomes a Confluence
  page. The file tree becomes the page tree, the first `# H1` becomes the title, Mermaid
  diagrams render as images, and links between pages are preserved.
- **Publish Markdown → GitHub Pages.** Generate a styled static site from your docs and
  push it to the repository's `gh-pages` branch — no conversion, rendered in the browser.
- **Two-way (Confluence).** Pull edits made on the platform back into your Markdown, with
  conflict detection when both sides changed.
- **Many destinations at once.** Configure as many outputs as you like; one **Sync** fans
  out across all of them.
- **A sidebar, not just commands.** The LoreMaster icon in the activity bar has three views:
  **Sync** (add a storage, sync, open the config), **Pages** and **Storages**.
- **See where everything stands.** The **Pages** view shows your Markdown as the tree it is on
  the storage, with an icon per page — new, synced, local changes, and (after a read-only
  *Check remote status*) remote changes or a conflict. Name pages by their title or their file
  name, whichever you prefer.
- **Change a storage without the YAML.** Edit a storage's prefix, direction, Mermaid mode,
  folders and more from the **Storages** view; the file keeps your comments.
- **Teach your AI agent the rules.** An MCP server tells an agent where a new page nests and
  how it is titled, and can preview and validate against your workspace. It is registered
  for the editor's agent automatically; **LoreMaster: Copy MCP Server Config** gives you the
  entry for Claude Desktop, Claude Code and other clients.
- **Generate pages from your project.** JUnit test reports, Go package documentation and OpenAPI
  descriptions become plain Markdown pages that sync like any other (`LoreMaster: Run generators`).
- **Leave things out.** `.gitignore`d Markdown is skipped by default (`skipGitignored`), and
  an `ignore` list takes gitignore-style patterns.

## Requirements

- VS Code **1.101** or later.
- The extension bundles its own engine — nothing else to install.
- For GitHub Pages: `git` on your `PATH` and push access to the repo (your existing
  credentials are used).

## Getting started

1. **Run `LoreMaster: Sync`** (Command Palette). On a fresh workspace it asks **where to
   sync** — tick the storages you want (Confluence, GitHub Pages) and configure each. Your
   choices are saved to `.lore-master.yaml` in the workspace root.
   - **Confluence:** first run `LoreMaster: Add Connection` (site URL + API token for
     Cloud, PAT for Data Center). Then Sync walks you through the space, parent page and
     title prefix.
   - **GitHub Pages:** confirm the repository (defaults to your `origin`) and branch
     (defaults to `gh-pages`).
2. **Sync.** `LoreMaster: Sync` publishes to every configured storage; it previews the
   plan before writing anything.

## Commands

| Command | What it does |
|---|---|
| **LoreMaster: Sync** | Sync every configured output (first run sets them up). |
| **LoreMaster: Sync to…** | Pick a subset of outputs to sync. |
| **LoreMaster: Sync Current File** | Sync just the active file to your Confluence outputs. |
| **LoreMaster: Publish to GitHub Pages** | Publish the site to the `gh-pages` branch. |
| **LoreMaster: Add Connection** | Sign in to a Confluence site. |
| **LoreMaster: Add sync storage** | Set up another destination for this workspace. |
| **LoreMaster: Edit storage** | Change one setting of a storage (also the pencil in the Storages view). |
| **LoreMaster: Open config** | Open `.lore-master.yaml`. |
| **LoreMaster: Check remote status** | Ask the platform where each page stands (read-only) and show it in the Pages view. |
| **LoreMaster: Show page titles or file names** | Switch how the Pages view names pages (setting `loreMaster.pages.label`). |
| **LoreMaster: Run generators** | Write Markdown from your test reports, Go packages and OpenAPI descriptions (configured under `generators:` in `.lore-master.yaml`); the next sync publishes it. |
| **LoreMaster: Copy MCP Server Config** | Copy the MCP server entry for an external AI client. |

## Configuration

Settings live in **`.lore-master.yaml`** at the workspace root (created for you on first
sync). Each entry under `outputs` is one destination:

```yaml
version: 1
outputs:
  - platform: confluence
    baseUrl: https://your-site.atlassian.net/wiki
    space: DOCS
    parentPageId: "123456"
    titlePrefix: Docs
    direction: two-way        # or to-platform
    content:
      - type: markdown
        roots: ["."]
        template: default
  - platform: github-pages
    direction: to-platform
    content:
      - type: markdown
        roots: ["docs"]
        template: default
```

Credentials are **never** stored in this file — they live in VS Code's secret store.

## Learn more

- Source, docs and issues: **https://github.com/LoreMaster-App/LoreMaster**
