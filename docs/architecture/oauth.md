# OAuth 2.0 sign-in: Cloud stays API-token, Data Center gets PKCE

Decided 2026-10-03 for epic E7 (#8); resolves the Cloud decision #41. Re-open if the
product leaves preview and demand changes the trade-off.

## The constraint

Confluence **Cloud** OAuth 2.0 (3LO) requires a `client_secret` and offers **no PKCE**
for public clients (an open Atlassian feature request). A distributed extension cannot
embed a secret, so it cannot complete the Cloud flow by itself. Only two shapes could:

- a **hosted broker** that holds the secret and exchanges the code (what Atlassian's own
  VS Code extension does) — permanent infrastructure, uptime, and it becomes the
  trust and privacy intermediary for every user's Confluence auth;
- **user-supplied app credentials** — each user registers a 3LO app and pastes a client
  id and secret, heavier UX than the API token it would sit beside.

**Data Center ≥ 7.17** is different: its OAuth 2.0 provider supports authorization code
**with PKCE**, so a public client needs no secret. An admin creates the incoming link and
hands out the client id.

## Decision

- **Cloud is API-token only.** Email + API token (Basic) stays the Cloud credential. We
  ship **no broker** and do **not** ask users to register a 3LO app.
- **Data Center gets OAuth** (#42): authorization code + PKCE, loopback redirect on
  `http://127.0.0.1:<port>/callback` (RFC 8252 — `127.0.0.1`, not `localhost`), tokens in
  the editor's secret store, silent refresh on 401, and a typed `ReauthRequired` on
  revocation. The engine returns the tokens in the `session/open` response and never
  persists them.

## Why

- It keeps the project's posture: **no infrastructure, no stored secrets** — the same
  reason Marketplace publishing uses Microsoft Entra ID (OIDC), not a stored PAT. A broker
  is a large, permanent, security-sensitive liability for a solo-maintained **preview**.
- Cloud already has a clean path. A user-supplied 3LO app is *worse* UX than email + API
  token — it still pastes a credential, and adds app registration on top — so it buys
  nothing over what v1 ships.
- Data Center genuinely gains: PKCE removes the secret problem, and refresh tokens beat a
  long-lived PAT. That is where OAuth earns its keep, so that is where it ships.

## Revisit when

The product leaves preview **and** there is real demand for one-click Cloud sign-in. Then
a hosted broker becomes worth its cost; this decision does not foreclose it.
