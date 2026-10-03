# Confluence tenant for the dogfood sync

Lore Master syncs its own `docs/` to Confluence from CI (#66, #67), which is both a demo
and a live test against a real site. This is how the tenant and its secrets are set up.
Nothing here is a secret; the secrets live only in GitHub Actions.

## One-time setup (maintainer)

1. **A Cloud site and space.** Use a free Confluence Cloud site (or an existing one).
   Create a space with key **`LORE`** (or any key — set `CONFLUENCE_SPACE` below to match).
2. **A parent page.** In that space, create a page titled **`Lore Master`**. The dogfood
   nests everything under it; it is found by title, so no page id is configured.
3. **A bot credential.** Mint an API token for the account the sync signs in as
   (ideally a dedicated bot user with write access to the space):
   <https://id.atlassian.com/manage-profile/security/api-tokens>.
4. **GitHub Actions secrets** (Settings → Secrets and variables → Actions → *Secrets*):
   - `CONFLUENCE_BASE_URL` — the site's wiki URL, e.g. `https://your-site.atlassian.net/wiki`.
   - `CONFLUENCE_EMAIL` — the account's email.
   - `CONFLUENCE_API_TOKEN` — the API token from step 3.
5. **Optional variables** (same screen → *Variables*), only if you deviate from the defaults:
   - `CONFLUENCE_SPACE` — the space key (default `LORE`).
   - `CONFLUENCE_PARENT_TITLE` — the parent page title (default `Lore Master`), used to find
     the parent when `CONFLUENCE_PARENT_PAGE_ID` is not set.
   - `CONFLUENCE_PARENT_PAGE_ID` — the parent page id, used as-is (skips the title lookup).
     Prefer this when you don't own the space or the parent title is awkward (e.g. an emoji).
   - `CONFLUENCE_TITLE_PREFIX` — the page title prefix (default `Lore Master`); pages are
     titled `<prefix>: <H1>`.
   - `CONFLUENCE_TITLE_COLLISION` — `adopt` (default) takes over an untracked page with the
     same title, fine in a space you own; set it to `fail` in a **shared** space so the sync
     refuses to touch pages it doesn't own.

Until `CONFLUENCE_BASE_URL` exists, the dogfood workflow **skips**, so `main` stays green.
Once the secrets are in place, the next push to `main` runs the sync. If the secrets exist
but the space or parent page is not there yet, the run **skips with a notice** rather than
failing — `main` stays green until the target is ready, then it syncs.

## Data Center (fixtures)

For the recorded Data Center fixtures (#30), add `CONFLUENCE_DC_BASE_URL` and
`CONFLUENCE_DC_PAT` (a personal access token, DC ≥ 7.9). A DC trial, or a local
`atlassian/confluence` Docker image with a trial license, works.

## How the dogfood behaves

- Runs on push to `main`, after the build, via `.github/workflows/dogfood.yml`.
- `tools/ci-sync` spawns the built engine, opens a session from the secrets, finds the
  parent page by title, writes an ephemeral `.lore-master.yaml` (so the tenant URL is
  never committed), plans the sync of `docs/`, **fails the job on any plan error or
  `conflict`**, then executes and prints the report as the job summary.
- **Annotations are not committed back.** CI uses `adopt` (title-collision) semantics, so
  the engine re-finds each page by its prefixed title on the next run; `docs/` therefore
  stays annotation-free in git. A re-run on the same commit is all `adopt`: it creates no
  duplicates and the content converges, but today it rewrites each page (bumping its
  Confluence version) rather than reporting `unchanged`, because `adopt` has no
  content-hash short-circuit yet (#137).
