# Lore Master — Claude Code Project Guide

## What this is

**Lore Master** is a family of editor plugins that gathers a project's lore. The first
feature syncs every Markdown file in an open workspace into a documentation platform,
Confluence first (Cloud, Data Center, and the end-of-life Server). More platforms and
more kinds of lore come later, which is why nothing here is named after Confluence or
Markdown.

The repository is still called `MarkDoc` until the maintainer renames it (#43).

## Two rules that govern all work here

1. **Every step is a GitHub issue before it is built.** The issues are the durable
   record; this file and `ROADMAP.md` only index them. A change without an issue is a
   change nobody can retrace.
2. **Every gap found in `@mnci/cli` is fixed in `russoedu/MoNecromanCi` and filed as an
   issue there** (label `found-by:lore-master`). This repo never works around mnci by
   hand without an issue saying so.

## Decisions already made (do not re-litigate)

| Decision | Choice | Where |
|---|---|---|
| Name | Lore Master; extension id `<publisher>.lore-master` | #43, #44 |
| Engine language | **Go**, one sidecar binary spoken to over JSON-RPC on stdio by every editor shell | `docs/architecture/engine-and-shells.md` |
| Distribution | No npm packages, no CLI product. Only the VS Code Marketplace (later the Visual Studio Marketplace). The binary ships inside each extension | E5 #6, #65 |
| Layout | Vertical feature slices in Go **and** TypeScript, even where no lint enforces it | `docs/architecture/vertical-feature-slices.md`, #47 |
| Page titles | `<titlePrefix>: <H1>`; prefix asked once at first sync, default = selected parent page's title | E3 #4 |
| Mermaid | Default `image`: code macro + SVG attachment rendered by the editor's webview; `code`, `html-macro`, `marketplace-macro` selectable | E6 #7 |
| Test runner (TS shell) | Jest | #45 |
| Auth v1 | Cloud: email + API token; DC: PAT; Server: PAT ≥ 7.9 else basic. OAuth is E7 (#8) | `docs/architecture/confluence-editions.md` |
| Monorepo tooling | `@mnci/cli` (Nx), GitHub Actions CI, merge commits only | #45, #50 |

## Layout

```
LoreMaster/                         one root go.mod, module lore-master (mnci derives it; MoNecromanCi#236)
├── apps/
│   ├── lore-master-engine/         go-app: JSON-RPC sidecar (main.go + slice packages)      E4 #5
│   └── lore-master-vscode/         VS Code extension, TypeScript (mnci vscode-extension)    E5 #6
├── libs/                           go-internal-lib each; import <module>/libs/<name>/<slice>
│   ├── markdown-workspace/         understand a folder of .md files                          E1 #2
│   ├── confluence-client/          talk to any Confluence edition                            E2 #3
│   └── documentation-sync/         reconcile a workspace with a platform (owns the ports)    E3 #4
├── tools/slicecheck/               Go test enforcing the file-role rules                     #47
├── docs/                           the project's own docs, synced by the tool (dogfood)     E8 #9
├── CLAUDE.md, ROADMAP.md
└── (files owned by mnci: nx.json, eslint.config.mnci.mjs, CI, .code-workspace, …)
```

Dependency direction, one way, enforced by the Go compiler:
`lore-master-engine` → `documentation-sync` → {`markdown-workspace`, `confluence-client`}.
`confluence-client` knows nothing about the engine; `documentation-sync` owns the
`DocumentationPlatform` and `DiagramRenderer` interfaces and a `confluenceplatform`
adapter package. A second platform is another adapter package.

## Slice rules in one screen

Capability → flat slice → role-suffixed files. Full ADR: `docs/architecture/vertical-feature-slices.md`.

- **Go**: a slice is one package directory directly under the project root. Exported
  identifiers are the barrel. A lib root holds only `doc.go`; an app root only `main.go`
  (+ `main_test.go`). Files are `<snake>_<role>.go`, tests `<snake>_<role>_test.go`, test
  data in `testdata/`. Package name = directory name, lowercase, no separators
  (`documenttree`). No sub-packages inside a slice. No `util`, `helpers`, `common`,
  `shared`, `internal`, `lib` packages.
- **TypeScript** (`apps/lore-master-vscode/src`): `<kebab>.<role>.ts`, one `index.ts` per
  slice, only `main.ts` at the root (`extension.ts` is rejected by the lint), tests
  `<name>.spec.ts`. Enforced by `@mnci/eslint-config`'s `verticalSlices` rules.
- Roles, both languages: `handler use-case algorithm policy model contract mapper
  validator repository client store error config enum`.
- Place a file in the slice whose outcome breaks if the file is deleted.

## How the sync works (the short version)

1. Discover `.md` files (roots/excludes from `lore-master.json`), parse with goldmark,
   title = first H1, build the tree: explicit `parent:` → dotted filename
   (`readme.architecture.md` under `readme.md`) → directory index (`README.md`) → the
   selected Confluence parent.
2. Plan: per file compare the `<!-- lore-master … -->` annotation (page id, version,
   content hash, attachment hashes) with the file and the remote page →
   `create | update | move | rename_title | adopt | unchanged | conflict | orphan`.
3. Execute parents-first: render Mermaid through the editor, upload changed attachments,
   create/update/move the page in storage format; links are `ri:page` by the target's
   prefixed title, known locally, so cycles resolve on the first pass.
4. Write annotations back only to files whose page changed. Report.

## Working in this repo

```bash
npm run lint      # ESLint (TS shell) + golangci-lint (Go) + tools/slicecheck
npm run test      # jest + go test ./...
npm run build     # engine binaries + extension bundle
npm run format    # eslint --fix (there is no Prettier)
```

- Go import paths are `lore-master/libs/<lib>/<slice>`. The module name comes
  from the npm scope (MoNecromanCi#236); fine for an app nobody `go get`s.
- Until MoNecromanCi#228 (`--registry none`) ships, two known reds are expected:
  `mnci doctor` flags the unused `NODE_AUTH_TOKEN` line in `.npmrc`, and
  `npm run release:preview` errors because nothing matches `release.projects`
  yet (CI's release step skips that case by itself).
- The workspace was generated from MoNecromanCi's unreleased branch build
  (`node <MoNecromanCi>/packages/cli/dist/cli.js`); use the published
  `npx @mnci/cli` once MoNecromanCi#227/#234/#235 are released.
- Before committing: `npm run format`, then `git diff` (mnci-owned files change on
  `mnci upgrade`; review them).
- Conventional commits are enforced by commitlint; `nx release` versions the extension
  from them. Merge PRs with a **merge commit**, never squash or rebase (#50).
- Secrets never touch this repo or `lore-master.json`; they live in the editor's secret
  store and travel to the engine per session over stdio.

## Where to look

- `ROADMAP.md` — every epic, decision and mnci dependency with links and status.
- `docs/architecture/engine-and-shells.md` — why one Go sidecar, the RPC surface,
  rejected alternatives.
- `docs/architecture/confluence-editions.md` — the verified API facts per edition.
- `russoedu/MoNecromanCi` issues #225–#233 — the mnci work this project waits on.
