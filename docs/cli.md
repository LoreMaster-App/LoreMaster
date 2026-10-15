# Command line

The engine binary that ships inside the extension is also a command line, for CI jobs and git
hooks. It runs the same method handlers the editors call, so `sync` plans and writes exactly
what the same sync from the editor would.

## Getting the binary in CI

Every release of the extension carries the engine for six platforms, stamped with the same
version, plus a checksum file. They are on the
[GitHub Releases](https://github.com/LoreMaster-App/LoreMaster/releases) page, under the tag
`lore-master-vscode@<version>`:

```
VERSION=0.14.10
BASE="https://github.com/LoreMaster-App/LoreMaster/releases/download/lore-master-vscode%40$VERSION"
curl -fsSLO "$BASE/go-app-lore-master-engine-linux-amd64.zip"
curl -fsSLO "$BASE/lore-master-engine-$VERSION-sha256.txt"
sha256sum --check --ignore-missing "lore-master-engine-$VERSION-sha256.txt"
unzip go-app-lore-master-engine-linux-amd64.zip
```

The zips are `go-app-lore-master-engine-<os>-<arch>.zip` for `windows`, `linux` and `darwin`
on `amd64` and `arm64`. Pin `VERSION`: the CLI's flags follow the extension's versions.

## Commands

```
lore-master-engine generate [--workspace DIR] [--generator N]... [--prepare] [--json]
lore-master-engine tree     [--workspace DIR] [--output N|NAME]... [--json]
lore-master-engine sync     [--workspace DIR] [--output N|NAME]... [--scope PATH]... [--generate] [--prepare]
                            [--yes] [--dry-run] [--force] [--prune] [--json]
lore-master-engine watch    [--workspace DIR] [--output N|NAME]... [--yes] [--force]
                            [--debounce 2s] [--poll 1s]
lore-master-engine pages build   --out DIR [--workspace DIR] [--output N|NAME] [--json]
lore-master-engine pages publish [--workspace DIR] [--output N|NAME] [--json]
lore-master-engine pages check   [--workspace DIR] [--output N|NAME] [--exit-code] [--json]
lore-master-engine rollback [--workspace DIR] [--journal ID] [--list] [--yes] [--force] [--json]
lore-master-engine version
```

- `generate` runs the generators of the `loreMaster.generators` setting. `--prepare` runs each
  generator's `prepare` command first (the build or test run that produces what it reads; see
  [generators](generators.md#producing-what-a-generator-reads-prepare)). Without it no command runs.
- `tree` shows the page tree each output would sync, with each page's local status.
- `rollback` undoes a sync, see [Undo a sync](#undo-a-sync).
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

`--output` takes an output's **name** (any case) or its position in the outputs list, and can be
repeated. Names are what the tree, the sync and the settings form show, so with two Confluences
(`"Technical documentation"` and `"User manuals"`) you can run
`lore-master-engine sync --output "User manuals"`.
- `pages publish` pushes the same site to the output's branch (default `gh-pages`).
- `pages check` says whether a publish would change anything, and lists the files it would add,
  modify or remove, without publishing. `--exit-code` exits 2 when the published site is out of
  date, so a pipeline can fail on stale docs. The VS Code **Check remote status** button does the
  same for a GitHub Pages output.
- `watch` keeps the storage up to date as files change, regenerating and syncing only what changed. See [watch mode](watch.md).

Outputs and generators are numbered by position in the settings.

Every command also takes `--settings JSON|@FILE` (settings that win over every settings file, the
way to configure a CI job with no settings file) and `--workspace-file FILE` (a `.code-workspace`
file whose settings sit between the user's and the folder's). See
[configuration](configuration.md#overrides-for-ci). `--json` prints one
document on stdout; progress lines always go to stderr.

## The CI session

**LoreMaster: Add to your CI pipeline** (Command Palette) builds the pipeline from the settings you
use locally, so you do not have to write it from this page:

1. It reads the effective settings: outputs, generators and their `prepare` commands.
2. It looks for the CI the repository already uses (`.github/workflows`, `azure-pipelines.yml`,
   `.azure-pipelines/`) and asks which one when it finds none or several. GitHub Actions and Azure
   Pipelines are covered; for another CI, use the [shell recipe above](#getting-the-binary-in-ci).
3. It opens a guide with the secrets to create (named from your Confluence edition: Cloud sites on
   `atlassian.net` get an email and a token, others a personal access token), the settings to commit
   when they live only in your user settings, and the pipeline itself.
4. It offers **Copy pipeline** or **Add** the file (`.github/workflows/lore-master.yml` or
   `.azure-pipelines/lore-master.yml`). It writes its own file and never edits a pipeline of yours; a
   file already there with other content is replaced only when you say so.

The pipeline downloads the engine of your extension's version and checks its checksum, runs
`sync --generate --yes` (with `--prepare` when a generator has a `prepare` command) and, for
Confluence outputs, commits `lore-master.json` back with `[skip ci]`.

## Undo a sync

Each Confluence sync that changes something is journalled in the user's cache folder (never in the
repository): the pages it created, the pages it updated with their earlier version, and the files
it rewrote with their earlier content.

```
lore-master-engine rollback --list                 # the syncs that can be undone, newest first
lore-master-engine rollback                        # shows what undoing the latest would do
lore-master-engine rollback --yes                  # does it
lore-master-engine rollback --journal ID --yes     # a specific one
```

Created pages are deleted (children first), updated pages are put back as they were and files get
their earlier content back. Without `--yes` nothing changes. A page or file edited after the sync is
skipped and reported, and the command exits with code 2; `--force` discards those edits too. It signs
in with the same environment variables as `sync`.

## Credentials

From the environment, never from a file:

| Variables | For |
|---|---|
| `LORE_MASTER_EMAIL` + `LORE_MASTER_TOKEN` | Confluence Cloud |
| `LORE_MASTER_PAT` | Confluence Data Center, Server 7.9+ |
| `LORE_MASTER_USER` + `LORE_MASTER_PASSWORD` | Confluence Server (basic) |

GitHub Pages outputs need none: git carries the credentials.

### Remember a connection instead

On your own machine you can keep the connection in the operating system's keychain (Windows
Credential Manager, macOS Keychain, the Linux Secret Service) instead of exporting the variables
in every shell:

```
LORE_MASTER_EMAIL=me@acme.com LORE_MASTER_TOKEN=... 
  lore-master-engine auth login --base-url https://acme.atlassian.net/wiki
lore-master-engine auth status        # what is remembered (never the tokens)
lore-master-engine auth logout --base-url https://acme.atlassian.net/wiki
```

`login` signs in first, so a wrong token is refused and not remembered. `sync` and `rollback` use
the remembered connection when the variables are not set; **the variables always win**, so a
pipeline's secret is never replaced by what a developer's machine remembered. The VS Code setting
`loreMaster.credentials.useKeychain` shares the same keychain with the extension.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Done |
| 1 | Failed: bad settings, refused credential, a generator or page failed |
| 2 | Stopped before changing the platform: plan errors, or pages edited on the platform and no `--force` |
| 64 | Wrong command line |

## Limits

Mermaid diagrams are drawn by the engine in a headless Chrome, Chromium or Edge, which it finds on
PATH or in the usual install folders (set `LORE_MASTER_BROWSER` to point at one). With none
installed they stay code blocks, with the usual warning, unless an editor is attached to draw
them. `LORE_MASTER_MERMAID=editor` makes the engine leave Mermaid to the editor even when it has a
browser. D2 diagrams are drawn by the engine itself, with no browser, so they are images from the
command line too.

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
