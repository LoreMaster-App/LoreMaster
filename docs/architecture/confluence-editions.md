# Confluence editions: the facts the design rests on

Verified 2026-10-01 while planning; re-verify before relying on a row that has aged.
Issues: #3 (client), #8 (OAuth), #7 (Mermaid).

| Fact | Consequence |
|---|---|
| Cloud REST **v2** has spaces and pages but **no attachment upload**. Upload is v1 `POST /rest/api/content/{id}/child/attachment` (multipart, `X-Atlassian-Token: nocheck`) on every edition | One attachment use case using v1 everywhere; pages use v2 on Cloud and v1 on DC/Server, split only inside `pagecontent/page_api_client.go` |
| **Storage format** (XHTML with `ac:`/`ri:` elements) is accepted by all editions; ADF is Cloud-only | One Markdown → storage-format mapper; no ADF |
| Page **titles are unique per space**; a clash is HTTP 400 | Every page is `<titlePrefix>: <H1>`; local and remote duplicate checks before any upload; `titleCollision: adopt | fail` |
| `ri:page` links resolve **by title at render time** | Cross-links are emitted on the first upload with locally-known titles; cycles need no second pass. `linkMode: id` (opt-in) is the only case that re-renders after creation |
| **Cloud auth**: email + API token as Basic. **Data Center ≥ 7.9**: personal access token as `Authorization: Bearer`. **Server** (end of life Feb 2024): PAT if ≥ 7.9, else Basic | Credential kinds `apitoken`, `pat`, `basic`; `Supports(edition, credential)` refuses the wrong pairing |
| **Cloud OAuth 2.0 (3LO)** requires a `client_secret`; no PKCE for public clients (open Atlassian feature request) | A shipped extension cannot complete the flow alone: hosted broker, user-supplied app credentials, or API-token-only (decision #41) |
| **Data Center ≥ 7.17** has an OAuth 2.0 provider supporting authorization code **with PKCE**; an admin creates the incoming link | DC sign-in is feasible (#42); loopback redirect on `127.0.0.1`, not `localhost` |
| No native **Mermaid**. The HTML macro is gone on Cloud and admin-disabled by default on DC | `mermaidMode`: `image` (default; editor renders SVG, uploaded as attachment), `code`, `html-macro` (where enabled), `marketplace-macro` |
| Rate limiting: 429 with `Retry-After`; occasional 5xx | Transport retries 429 per header, backs off on 5xx, never blindly retries a non-idempotent POST |
| XML comments containing `--` invalidate a storage document | The mapper never emits `--` inside a comment |
| Prior art: `kovetskiy/mark` uses `<!-- Key: value -->` headers, renders Mermaid to PNG/SVG, resolves `./x.md` links in a second pass | Validates the annotation design; our title-based links remove the second pass |
