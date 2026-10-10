# LoreMaster documentation

Start here, then dive into the architecture decisions.

## Using it

- [Getting started](getting-started.md) — install, connect, and run your first sync.
- [GitHub wiki](github-wiki.md) — publish the Markdown as the repository's wiki pages, beside
  or instead of a Pages site.
- [Generators](generators.md) — pages written from your test reports (and later source docs),
  synced like any other page.
- [A documentation agent](agent.md) — create an agent for Claude Code, VS Code or `AGENTS.md` that writes
  docs the way the sync expects.
- [Using the MCP server](mcp-server.md) — let an AI agent learn LoreMaster's nesting rules and
  check its files against them.

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
