# LoreMaster

![LoreMaster](https://raw.githubusercontent.com/LoreMaster-App/LoreMaster/main/assets/lore-master-concept-md.jpg)

**Your project already knows everything your team needs to read. LoreMaster publishes it.**

Keep your docs as plain Markdown in the repository. LoreMaster turns them into a clean page tree on
**Confluence** (Cloud, Data Center, Server), a styled **GitHub Pages** site or your **GitHub wiki**,
and keeps it in step on every run, from VS Code or unattended on CI. Add **generators** and the pages
nobody wants to write by hand write themselves: **test results, code coverage, API reference, Go,
TypeScript, Python, C# and Dart documentation, and a Storybook with real screenshots**.

![From your repository to every storage](https://raw.githubusercontent.com/LoreMaster-App/LoreMaster/main/assets/listing/how.png)

> **Preview.** The engine and its libraries are built and tested; this extension is published as a
> preview while the end-to-end sync is hardened.

## Highlights

- **Preview, then publish.** Every sync shows its plan first: create, update, move, rename, adopt,
  unchanged or conflict. A page someone edited on the platform since your last sync is a conflict,
  never overwritten.
- **The folder tree is the page tree.** The first `# H1` is the title; links, images and diagrams
  (Mermaid and D2) come along. Run it again and nothing changes unless a file did.
- **Generators write the pages nobody writes.** Test results, coverage, OpenAPI, Storybook, and
  code documentation for five languages, all as ordinary Markdown that syncs like any page.
- **Fresh as you type.** Watch mode regenerates and syncs only what changed.
- **The same engine on CI.** A command line and a GitHub Action for unattended runs.
- **Your AI agent learns the rules.** An MCP server places pages, validates them and plans
  reorganisations.

## Get started

1. Install **LoreMaster** and open a folder with Markdown. The engine is bundled; there is nothing
   else to install.
2. Run **LoreMaster: Sync** from the Command Palette. On a fresh workspace it asks where to publish:
   - **Confluence:** run **LoreMaster: Add Connection** first (site URL and API token for Cloud, a
     personal access token for Data Center), then Sync walks you through the space, parent page and
     title prefix.
   - **GitHub Pages / wiki:** confirm the repository (defaults to your `origin`) and the branch.
3. Review the plan and let it write.

Tokens live in VS Code's secret store, never in settings or in the repository.

## In the editor

![The VS Code sidebar and commands](https://raw.githubusercontent.com/LoreMaster-App/LoreMaster/main/assets/listing/editor.png)

The LoreMaster icon in the activity bar has four views:

- **Sync**: add a storage, sync everything or one file, generate and sync in one step.
- **Pages**: your Markdown as the tree it becomes, with an icon per page (new, synced, local changes,
  remote changes, conflict). Name pages by title or by file name.
- **Generators**: add, run, edit and remove generators, and see what each one did.
- **Storages**: change a prefix, direction or diagram mode without editing JSON; the file keeps your
  comments.

Also: **two-way** sync for Confluence (pull remote edits back into Markdown), **Sync Current File**,
**Check remote status** (read-only), and a **documentation agent** written for Claude Code, VS Code
or `AGENTS.md` that knows your nesting rules.

## Generators

![The generators](https://raw.githubusercontent.com/LoreMaster-App/LoreMaster/main/assets/listing/generators.png)

A generator reads something your project already produces and writes Markdown pages. **Leave the
type out and LoreMaster recognises the format.** If it cannot, it says *could not identify the
output, no parser available* instead of staying silent.

| Generator | Reads | Writes |
|---|---|---|
| `test-results` | JUnit XML | a page per suite, with failures and totals |
| `coverage` | Cobertura XML, LCOV | a page per folder with coverage bars |
| `openapi-docs` | OpenAPI 3 | operations by tag, parameters, schemas |
| `storybook-docs` | a Storybook build or URL | a page per component with real screenshots |
| `markdown-docs` | Markdown another tool wrote | the same pages, images included |
| `go-docs`, `ts-docs`, `python-docs`, `csharp-docs`, `dart-docs` | source code | a page per package, module or type |

```jsonc
// .vscode/settings.json
"loreMaster.generators": [
  { "prepare": "npm test", "input": ["reports/"], "output": "docs/tests", "title": "Test results" },
  { "source": "storybook-static", "prepare": "npm run build-storybook", "output": "docs/components" }
]
```

`prepare` is a command you choose; it runs only when you ask. The Storybook generator renders
stories in the Chrome, Chromium or Edge already on your machine.

## On CI

![The engine in CI](https://raw.githubusercontent.com/LoreMaster-App/LoreMaster/main/assets/listing/ci.png)

The engine is also a command line, shipped as a static binary for Linux, macOS and Windows on every
[release](https://github.com/LoreMaster-App/LoreMaster/releases), with a checksum file:

```bash
lore-master-engine generate --prepare            # reports, coverage, API, code -> Markdown
lore-master-engine sync --generate --yes         # ...and onto Confluence / the wiki
lore-master-engine pages build --out dist/docs   # a Pages site, no git, no network
lore-master-engine pages check --exit-code       # fail the build when the docs are stale
```

A **GitHub Action** (`LoreMaster-App/LoreMaster/actions/pages`) builds the site beside your web app in
one step.

## For your AI agent

The extension registers an **MCP server** with VS Code's agent. **LoreMaster: Copy MCP Server Config**
gives you the entry for Claude Desktop, Claude Code and other clients.

| Tool | The agent can |
|---|---|
| `loremaster_nesting_rules`, `preview_tree` | learn the rules and see the page tree |
| `place_document`, `validate_document` | place a new page and check a file |
| `plan_moves` | reorganise: what must move, and where each page will nest afterwards |
| `list_generators` | know which folders are generated and must not be edited by hand |
| `site_publishing_plan`, `preview_site` | publish the docs site beside an app deploy |

## Commands

| Command | What it does |
|---|---|
| **LoreMaster: Sync** | Sync every configured output (first run sets them up). |
| **LoreMaster: Sync to…** | Pick a subset of outputs to sync. |
| **LoreMaster: Sync Current File** | Sync just the active file. |
| **LoreMaster: Generate and sync documentation** | Run the generators, then sync. |
| **LoreMaster: Run generators** / **Add generator** | Write pages from reports, source and Storybook. |
| **LoreMaster: Toggle watch mode** | Keep generated pages and the storage fresh as you type. |
| **LoreMaster: Publish to GitHub Pages** | Publish the site to the `gh-pages` branch. |
| **LoreMaster: Add Connection** | Sign in to a Confluence site. |
| **LoreMaster: Add sync storage** / **Edit storage** | Set up or change a destination. |
| **LoreMaster: Check remote status** | Ask the platform where each page stands (read-only). |
| **LoreMaster: Create documentation agent** | Write an agent that follows your structure. |
| **LoreMaster: Copy MCP Server Config** | The MCP entry for an external AI client. |
| **LoreMaster: Open settings** | Settings, with every option described. |

## Configuration

Everything lives in VS Code's settings under `loreMaster.*`, layered **folder > workspace > user**, so
set your Confluence address once in User settings.

```jsonc
// .vscode/settings.json
{
  "loreMaster.confluence.baseUrl": "https://your-site.atlassian.net/wiki",
  "loreMaster.outputs": [
    {
      "platform": "confluence",
      "space": "DOCS",
      "parentPageId": "123456",
      "titlePrefix": "Docs",
      "direction": "two-way",
      "content": [{ "type": "markdown", "roots": ["."] }]
    },
    {
      "platform": "github-pages",
      "direction": "to-platform",
      "content": [{ "type": "markdown", "roots": ["docs"] }]
    }
  ]
}
```

`.gitignore`d Markdown is skipped by default, and an `ignore` list takes gitignore-style patterns.

## Requirements

- VS Code **1.101** or later.
- For GitHub Pages and the wiki: `git` on your `PATH` and push access (your existing credentials).
- For `storybook-docs`: Chrome, Chromium or Edge. For `ts-docs`, `python-docs`, `csharp-docs` and
  `dart-docs`: the documentation tool each one names, which it never installs for you.

## Learn more

- [Getting started](https://github.com/LoreMaster-App/LoreMaster/blob/main/docs/getting-started.md)
- [Generators](https://github.com/LoreMaster-App/LoreMaster/blob/main/docs/generators.md) ·
  [Command line](https://github.com/LoreMaster-App/LoreMaster/blob/main/docs/cli.md) ·
  [GitHub Action](https://github.com/LoreMaster-App/LoreMaster/blob/main/docs/github-action.md) ·
  [MCP server](https://github.com/LoreMaster-App/LoreMaster/blob/main/docs/mcp-server.md)
- Source and issues: **https://github.com/LoreMaster-App/LoreMaster**
