# Command line

The engine binary that ships inside the extension is also a command line, for CI jobs and git
hooks. It runs the same method handlers the editors call, so `sync` plans and writes exactly
what the same sync from the editor would.

```
lore-master-engine generate [--workspace DIR] [--generator N]... [--json]
lore-master-engine tree     [--workspace DIR] [--output N]... [--json]
lore-master-engine sync     [--workspace DIR] [--output N]... [--scope PATH]... [--generate]
                            [--yes] [--dry-run] [--force] [--prune] [--json]
lore-master-engine watch    [--workspace DIR] [--output N]... [--yes] [--force]
                            [--debounce 2s] [--poll 1s]
lore-master-engine pages build   --out DIR [--workspace DIR] [--output N] [--json]
lore-master-engine pages publish [--workspace DIR] [--output N] [--json]
lore-master-engine pages check   [--workspace DIR] [--output N] [--exit-code] [--json]
lore-master-engine version
```

- `generate` runs the generators of `.lore-master.yaml`.
- `tree` shows the page tree each output would sync, with each page's local status.
- `sync` plans each output. **Without `--yes` it only shows the plan.** `--generate` runs the
  generators first; `--force` overwrites pages edited on the platform; `--prune` trashes pages
  whose file is gone.

- `pages build`, `pages publish` and `pages check` also take a `github-wiki` output; see
  [GitHub wiki](github-wiki.md).
- `pages build` renders a GitHub Pages output's static site into `--out` with no git and no
  network. The folder is created, replaced when an earlier build wrote it (it carries a
  `.lore-master-site` marker), and **refused when it holds other files**, so a build can never
  wipe a web app that shares the folder. Use it when your pipeline deploys the folder itself
  (a Pages artifact, an object store); see [Docs beside another site](#docs-beside-another-site).
  `--output` is needed only when the settings have several GitHub Pages outputs.
- `pages publish` pushes the same site to the output's branch (default `gh-pages`).
- `pages check` says whether a publish would change anything, and lists the files it would add,
  modify or remove, without publishing. `--exit-code` exits 2 when the published site is out of
  date, so a pipeline can fail on stale docs. The VS Code **Check remote status** button does the
  same for a GitHub Pages output.
- `watch` keeps the storage up to date as files change, regenerating and syncing only what changed. See [watch mode](watch.md).

Outputs and generators are numbered by position in the settings file. `--json` prints one
document on stdout; progress lines always go to stderr.

## Credentials

From the environment, never from a file:

| Variables | For |
|---|---|
| `LORE_MASTER_EMAIL` + `LORE_MASTER_TOKEN` | Confluence Cloud |
| `LORE_MASTER_PAT` | Confluence Data Center, Server 7.9+ |
| `LORE_MASTER_USER` + `LORE_MASTER_PASSWORD` | Confluence Server (basic) |

GitHub Pages outputs need none: git carries the credentials.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Done |
| 1 | Failed: bad settings, refused credential, a generator or page failed |
| 2 | Stopped before changing the platform: plan errors, or pages edited on the platform and no `--force` |
| 64 | Wrong command line |

## Limits

Mermaid diagrams stay code blocks (the editor draws them; the command line has no browser), with
the usual warning.

```yaml
# GitHub Actions
- run: lore-master-engine sync --generate --yes
  env:
    LORE_MASTER_EMAIL: ${{ secrets.CONFLUENCE_EMAIL }}
    LORE_MASTER_TOKEN: ${{ secrets.CONFLUENCE_TOKEN }}
```

## Docs beside another site

GitHub Pages serves one site per repository, and a deploy replaces all of it. When the repository
already deploys something to Pages (a web app, say), build the docs into a subfolder of that
deploy instead of publishing a branch:

```yaml
# the Actions-source workflow that already builds the app
- run: npm run build                       # produces dist/
- run: lore-master-engine pages build --out dist/docs
- uses: actions/upload-pages-artifact@v4
  with:
    path: dist
```

The app stays at `/` and the docs appear at `/docs/`. Every link in the site is relative, so it
works under any subpath. `upload-pages-artifact` leaves dotfiles out, so the marker is not
deployed.

If the repository deploys from a **branch** instead (for example `gh-pages` written by another
tool), publish the docs into a folder of that branch with the output's `path` setting:

```yaml
outputs:
  - platform: github-pages
    direction: to-platform
    branch: gh-pages
    path: docs          # only this folder is replaced
    content:
      - type: markdown
        roots: ["."]
        template: default
```

The other deploy must keep what it did not write (`keep_files: true` for
`peaceiris/actions-gh-pages`). Publishing without `path` to a branch that holds another site is
refused and the branch is left untouched; a branch an earlier LoreMaster published to is
recognised and replaced as before.
