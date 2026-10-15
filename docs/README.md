# LoreMaster documentation

Start here, then dive into the architecture decisions.

## Using it

![From your repository to every storage](../assets/listing/how.png)

**Start here**

- [Getting started](getting-started.md): install, connect, and run your first sync.
- [Configuration](configuration.md): every `loreMaster.*` setting and how the layers combine.
- [Choosing what goes where](choosing-what-goes-where.md): send some pages to Pages, some to
  Confluence and some to the wiki, with exclude and include lists.
- [The `[children]` tag](children-tag.md): list a page's child pages with Confluence's children
  macro, or as links on a site or wiki.
- [Sync state](sync-state.md): the one `lore-master.json` that records which page each file became
- [Page templates](templates.md): lay out each output's pages with a built-in or your own
  template: a site shell, a wiki Markdown file or a Confluence page.

**Publish**

- [GitHub wiki](github-wiki.md): the Markdown as the repository's wiki pages, beside or instead
  of a Pages site.
- [GitHub Action](github-action.md): build the docs site in one step, beside a web app or onto
  a branch.

**Generate pages from what you already produce**

- [Generators](generators.md): test results, coverage, OpenAPI, Markdown imports, Storybook with
  real screenshots, and code documentation for Go, TypeScript, Python, C# and Dart.
- [Watch mode](watch.md): regenerate and sync only what changed, as you type.

**Automate**

- [Command line](cli.md): `generate`, `sync`, `tree`, `watch` and `pages` for CI and git hooks,
  with the engine binaries attached to every release.

**Work with AI agents**

- [A documentation agent](agent.md): create an agent for Claude Code, VS Code or `AGENTS.md`
  that writes docs the way the sync expects.
- [Using the MCP server](mcp-server.md): place pages, validate them, plan reorganisations and
  list the generated folders.

## Architecture (the decisions the design rests on)

- [Engine and shells](architecture/engine-and-shells.md) — one Go sidecar over JSON-RPC,
  thin editor shells, and the alternatives rejected.
- [Confluence editions](architecture/confluence-editions.md) — the verified per-edition API
  facts the client is built on.
- [The MCP server](architecture/mcp-server.md) — why it lives in the engine, the six tools,
  and the "one source of truth" invariant.
- [Vertical feature slices](architecture/vertical-feature-slices.md) — how the Go and
  TypeScript code is organised.

## Maintaining it

- [Confluence tenant for recorded fixtures](contributing/confluence-tenant.md) — the
  credentials and setup for recording the #29/#30 fixtures.

See also the project [`README.md`](../README.md), [`ROADMAP.md`](../ROADMAP.md), and
[`CLAUDE.md`](../CLAUDE.md) at the repository root.
