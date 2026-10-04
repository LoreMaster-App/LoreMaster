# Lore Master documentation

Start here, then dive into the architecture decisions.

## Using it

- [Getting started](getting-started.md) — install, connect, and run your first sync.
- [Signing in with OAuth (Data Center)](contributing/confluence-oauth.md) — set up and use
  OAuth 2.0 sign-in.

## Architecture (the decisions the design rests on)

- [Engine and shells](architecture/engine-and-shells.md) — one Go sidecar over JSON-RPC,
  thin editor shells, and the alternatives rejected.
- [Confluence editions](architecture/confluence-editions.md) — the verified per-edition API
  facts the client is built on.
- [OAuth decision](architecture/oauth.md) — why Cloud stays API-token only and OAuth ships
  Data Center only.
- [Vertical feature slices](architecture/vertical-feature-slices.md) — how the Go and
  TypeScript code is organised.

## Maintaining it

- [Confluence tenant for recorded fixtures](contributing/confluence-tenant.md) — the
  credentials and setup for recording the #29/#30 fixtures.

See also the project [`README.md`](../README.md), [`ROADMAP.md`](../ROADMAP.md), and
[`CLAUDE.md`](../CLAUDE.md) at the repository root.
