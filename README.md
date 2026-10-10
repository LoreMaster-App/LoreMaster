# LoreMaster

![LoreMaster](./assets/lore-master-concept-md.jpg)

**LoreMaster** gathers a project's lore and publishes it where your team already reads.
Its first feature syncs every Markdown file in your workspace into a documentation
platform — **Confluence** first (Cloud, Data Center, and the end-of-life Server) — as a
clean page tree, and keeps it in sync on every run.

It ships as an editor extension (**VS Code** today, Visual Studio next) around a single
engine, so the same sync behaves identically in every editor. The project is deliberately
not named after Confluence or Markdown: more platforms and more kinds of lore come later.

> **Status: preview.** The engine and its libraries are built and tested; the VS Code
> extension is published to the Marketplace as a preview (`LoreMaster.loremaster`) while the
> end-to-end sync is hardened. See [`ROADMAP.md`](ROADMAP.md).

---

## What it does

- **Discovers** the Markdown in your workspace and turns it into a page **tree** —
  nesting, cross-links, images, and Mermaid diagrams included.
- **Plans** before it writes: it shows you exactly what will be created, updated, moved,
  renamed, adopted, left unchanged, or flagged as a conflict — and refuses to overwrite a
  page someone edited on the platform since the last sync.
- **Executes** parents-first, so links resolve on the first pass without a second write.
- **Shows where everything stands** in a **Pages** view: your Markdown as the tree it is on the storage,
  with an icon per page — new, synced, local changes, remote changes, conflict — and a title or file name label, as you prefer.
- **Creates a documentation agent** for Claude Code, VS Code or `AGENTS.md`, taught the nesting rules and
  your workspace's layout. See [`docs/agent.md`](docs/agent.md).
- **Generates pages from your artifacts** — JUnit test reports, Go package docs and OpenAPI descriptions — as
  plain Markdown that syncs like any other page, managed from a **Generators** view. See [`docs/generators.md`](docs/generators.md).
- **Command line.** The engine also runs `generate`, `sync` and `tree` unattended for CI. See [`docs/cli.md`](docs/cli.md).
- **Watch mode.** Edit a source comment and the page updates within seconds, without a full sync: `lore-master-engine watch` or the editor's *Toggle watch mode*. See [`docs/watch.md`](docs/watch.md).
- **Teaches your AI agent the rules** through an [MCP server](docs/mcp-server.md): the agent
  asks LoreMaster where a new page nests and how it is titled, and can preview and validate
  against your actual workspace.
- **Is idempotent**: run it again and nothing changes unless a file did. It tracks each
  page with a small annotation written back into the file (page id, version, content and
  attachment hashes), so it never creates duplicates and only writes what actually moved.

## How the sync works

1. **Discover & build the tree.** Find `.md` files (roots and excludes from
   `.lore-master.yaml`), parse them with goldmark, take the title from the first `# H1`,
   and build the hierarchy. A page's parent is decided, in order, by: an explicit
   `parent:` in the file's annotation → a **dotted filename**
   (`readme.architecture.md` nests under `readme.md`) → a **directory index**
   (`README.md`) → the Confluence parent page you selected.
2. **Plan.** For each file, compare the `<!-- lore-master … -->` annotation against the
   file and the remote page, and choose one action:
   `create · update · move · rename_title · adopt · unchanged · conflict · orphan`.
3. **Execute.** Parents first, so a new page's id is known before its children are made:
   render Mermaid through the editor, upload changed attachments, create/update the page in
   Confluence **storage format**. Links are emitted as `ri:page` references by the target's
   prefixed title, so cycles resolve without a second pass (an opt-in `linkMode: id` is the
   only case that re-renders).
4. **Write back & report.** Update the annotation only in files whose page changed, then
   report every outcome.

### Page titles

Every page is titled `<titlePrefix>: <H1>`. The prefix is asked once on the first sync
(defaulting to the selected parent page's title). Confluence titles are unique per space,
so a clash is handled by `titleCollision`: `adopt` takes over an untracked page with the
same title (fine in a space you own), `fail` refuses to touch pages the sync did not make
(use this in a shared space).

### Mermaid diagrams

Confluence has no native Mermaid, so `mermaidMode` chooses how diagrams appear:

- `image` (default) — the editor's webview renders the diagram to SVG, which is uploaded as
  an attachment and shown with the Mermaid source kept in a code macro. Portable across
  platforms.
- `code` — leave it as a fenced code block.
- `html-macro` / `marketplace-macro` — where your site enables them.

## Authentication

Credentials never touch this repository or `.lore-master.yaml`. They live in the editor's
**secret store** and travel to the engine per session over stdio.

| Edition | Credential |
|---|---|
| **Cloud** | Atlassian account email + API token |
| **Data Center** | personal access token (≥ 7.9) |
| **Server** (EOL) | personal access token (≥ 7.9), else user name + password |

## Configuration — `.lore-master.yaml`

The sync is driven by a per-workspace YAML file (written for you on the first sync):

```yaml
version: 1
skipGitignored: true         # default; leave out .md files your .gitignore files ignore
ignore: []                    # gitignore-style patterns every output leaves out, e.g. [drafts/**, NOTES.md]
outputs:
  - platform: confluence
    baseUrl: https://your-site.atlassian.net/wiki
    space: DOCS
    parentPageId: "123456"      # the page everything nests under
    titlePrefix: "LoreMaster"  # pages are titled "<prefix>: <H1>"
    direction: to-platform      # to-platform (push) | two-way (also pull remote edits back)
    mermaidMode: image          # image | code | html-macro | marketplace-macro
    titleCollision: adopt       # adopt | fail
    linkMode: title             # title | id
    content:
      - type: markdown
        roots: [docs]           # folders to sync
        excludes: []
        template: default
```

`skipGitignored` (default `true`) skips Markdown that the workspace's `.gitignore` files (nested ones
included) ignore — usually drafts, vendored copies or build output; set it to `false` to sync them. `ignore`
is a list of gitignore-style patterns, matched against workspace-relative paths, that every output leaves out
on top of its own `content[].excludes`. `node_modules`, `.git`, `dist`, `out-tsc`, `coverage` and `.venv` are
always skipped.

`direction: two-way` also pulls edits made on the platform back into the Markdown (a page
changed on both sides is reported as a conflict, never merged). Other reserved values (more
content types, templates) are present in the schema but refused until built.

## Architecture

LoreMaster is **one Go engine** — a sidecar binary spoken to over **JSON-RPC on stdio** —
wrapped by a thin editor shell per editor. The shell handles UI, secrets, and the browser;
the engine does all the Confluence work. This keeps the sync identical everywhere and the
shells small. The rationale and rejected alternatives are in
[`docs/architecture/engine-and-shells.md`](docs/architecture/engine-and-shells.md).

Dependency direction (enforced by the Go compiler):

```
lore-master-engine → documentation-sync → { markdown-workspace, confluence-client }
```

`documentation-sync` owns the `DocumentationPlatform` and `DiagramRenderer` ports and a
`confluenceplatform` adapter; a second platform is just another adapter.
`confluence-client` knows nothing about the engine.

Code is organised in **vertical feature slices** (capability → cohesive subfeature →
role-suffixed files) in both Go and TypeScript —
[`docs/architecture/vertical-feature-slices.md`](docs/architecture/vertical-feature-slices.md).

## Repository layout

```
LoreMaster/                         one root go.mod, module "lore-master"
├── apps/
│   ├── lore-master-engine/         Go: the JSON-RPC sidecar (main.go + slice packages)
│   └── lore-master-vscode/         VS Code extension (TypeScript)
├── libs/                           Go internal libraries
│   ├── markdown-workspace/         understand a folder of .md files
│   ├── confluence-client/          talk to any Confluence edition
│   └── documentation-sync/         reconcile a workspace with a platform (owns the ports)
├── tools/slicecheck/               Go test enforcing the file-role rules
├── docs/                           architecture and contributing docs
├── CLAUDE.md, ROADMAP.md
└── (files owned by @mnci/cli: nx.json, eslint config, CI, …)
```

## Developing

Prerequisites: **Node** (for the Nx workspace and the TypeScript shell) and **Go** (for the
engine and libraries). The monorepo is managed with
[`@mnci/cli`](https://github.com/russoedu/MoNecromanCi) (Nx under the hood).

```bash
npm run lint      # ESLint (TS) + golangci-lint (Go) + tools/slicecheck
npm run test      # jest + go test ./...
npm run build     # engine binaries + extension bundle
npm run format    # eslint --fix (there is no Prettier)
```

Debug the extension with the **`lore-master-vscode: debug`** launch entry (an Extension
Development Host). Package the VSIX and stamped engine binaries with
`npx nx run lore-master-vscode:package`.

### Testing

Every library is unit-tested. The whole sync is additionally covered end to end, with **no
real tenant and no network**, by a hermetic integration test
(`libs/documentation-sync/remotesync`): it drives the real adapter and HTTP client through
the real plan/execute use cases against an in-process fake Confluence, asserting page
creation, annotation write-back, an idempotent re-run, and page adoption.

## Contributing

- **Every step is a GitHub issue before it is built** — the issues are the durable record;
  `CLAUDE.md` and `ROADMAP.md` only index them.
- **Conventional commits** (enforced by commitlint); **merge commits only** (never squash
  or rebase).
- Releases are automated: a merge to `main` versions the extension from the commit history
  and publishes it to the Marketplace through Microsoft Entra ID (OIDC — no stored secret).
  The same packages go to Open VSX (VSCodium, Cursor, Windsurf, Gitpod) when the `OVSX_PAT`
  secret exists; create the `LoreMaster` namespace once with `npx ovsx create-namespace`.

See [`docs/`](docs/) for architecture and contributing guides, and
[`ROADMAP.md`](ROADMAP.md) for every epic and its issues.

## License

See [`LICENSE`](LICENSE).
