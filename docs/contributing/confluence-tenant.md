# Confluence tenant for recorded fixtures

The sync is covered end to end by a Go integration test that drives it against a **fake**
Confluence (an in-process `httptest` server), so the test suite needs no real tenant and
writes no real pages. See `libs/documentation-sync` for that test.

A real tenant is still useful for one thing: **recording the fixtures** (#29 Cloud, #30
Data Center) that let the fake replay real API shapes. That is a maintainer task run
locally with credentials in the environment — never from CI, and never against a space
you do not own. Nothing here is a secret; the credentials live only in your shell.

## Cloud (fixtures, #29)

1. A Confluence Cloud site and a space you own.
2. An API token for the account that records: <https://id.atlassian.com/manage-profile/security/api-tokens>.
3. Export for the recorder:
   - `CONFLUENCE_BASE_URL` — the site's wiki URL, e.g. `https://your-site.atlassian.net/wiki`.
   - `CONFLUENCE_EMAIL` — the account's email.
   - `CONFLUENCE_API_TOKEN` — the token from step 2.

## Data Center (fixtures, #30)

A DC trial, or a local `atlassian/confluence` Docker image with a trial license, works.
Export `CONFLUENCE_DC_BASE_URL` and `CONFLUENCE_DC_PAT` (a personal access token, DC ≥ 7.9).
