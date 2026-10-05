# Signing in to Confluence Data Center with OAuth

LoreMaster can sign in to **Confluence Data Center** with OAuth 2.0 (authorization code +
PKCE) instead of a personal access token. **Confluence Cloud stays API-token only** — the
reasoning is in [`../architecture/oauth.md`](../architecture/oauth.md). This guide is how to
set it up and what happens under the hood.

Requires **Data Center 7.17 or later**, which is the first release with the OAuth 2.0
provider that supports PKCE (so a distributed extension needs no client secret).

## One-time admin setup

A site admin creates an **incoming OAuth 2.0 link** on the Data Center site and gives each
user its **client id**. The exact screens vary by version — see Atlassian's own
documentation for "OAuth 2.0 provider" / "incoming application links" on Data Center — but
two things matter for LoreMaster:

- **Redirect / callback.** The extension completes the flow on a **loopback** address,
  `http://127.0.0.1:<port>/callback` (RFC 8252; `127.0.0.1`, not `localhost`, because some
  allow-lists reject the name). The port is chosen at sign-in time, so the link must permit
  the loopback redirect rather than a single fixed port.
- **Scopes.** Grant the link the scopes a sync needs (reading and writing pages and
  attachments — commonly `WRITE`). The user enters the scope during sign-in; it must match
  what the link allows.

The client **secret** is not needed and is never entered into the editor — PKCE replaces it.

## Signing in (in VS Code)

Two ways in, both reaching the same connection:

- the **Accounts menu** (bottom-left in VS Code) → *Sign in with Confluence (LoreMaster)*, or
- the **LoreMaster: Add Connection** command.

Then:

1. Enter the site URL (any page URL of the site works); the extension detects Data Center.
2. Choose **OAuth sign-in**.
3. Enter the **client id** from your admin, and the **scope** if your link requires one
   (leave it blank to request none).
4. Your browser opens the site's authorize page — approve the request.
5. You are signed in. The account shows in the Accounts menu, and syncs use it.

## What happens under the hood

- The engine generates a PKCE verifier and challenge, opens
  `/rest/oauth2/latest/authorize` in your browser through the editor, and waits on the
  loopback port for the redirect (the `state` is checked, so a forged redirect is rejected).
- It exchanges the code at `/rest/oauth2/latest/token` for an **access token** and a
  **refresh token**, and hands them back to the editor.
- **Tokens live only in the editor's secret store** — never in the repository or in
  `.lore-master.yaml` — and travel to the engine per session over stdio. The engine keeps
  none of its own.
- When a stored access token has expired, the next session **refreshes** it from the refresh
  token automatically and stores the renewed pair.
- If the refresh token has been **revoked or expired**, the sign-in can no longer be renewed:
  LoreMaster reports that you need to sign in again rather than failing silently. Sign in
  once more from the Accounts menu or the command.

## Signing out

Remove the account from the **Accounts menu** (*Sign out*), or remove the connection the same
way you added it. That deletes the stored tokens for the site.
