# LoreMaster documentation

Start here, then dive into the architecture decisions.

## Using it

- [Getting started](getting-started.md) — install, connect, and run your first sync.
- [Using the MCP server](mcp-server.md) — let an AI agent learn LoreMaster's nesting rules and
  check its files against them.

## Architecture (the decisions the design rests on)

- [Engine and shells](architecture/engine-and-shells.md) — one Go sidecar over JSON-RPC,
  thin editor shells, and the alternatives rejected.
- [Confluence editions](architecture/confluence-editions.md) — the verified per-edition API
  facts the client is built on.
- [The MCP server](architecture/mcp-server.md) — why it lives in the engine, the four tools,
  and the "one source of truth" invariant.
- [Vertical feature slices](architecture/vertical-feature-slices.md) — how the Go and
  TypeScript code is organised.

## Maintaining it

- [Confluence tenant for recorded fixtures](contributing/confluence-tenant.md) — the
  credentials and setup for recording the #29/#30 fixtures.

See also the project [`README.md`](../README.md), [`ROADMAP.md`](../ROADMAP.md), and
[`CLAUDE.md`](../CLAUDE.md) at the repository root.
