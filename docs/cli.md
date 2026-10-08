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
lore-master-engine version
```

- `generate` runs the generators of `.lore-master.yaml`.
- `tree` shows the page tree each output would sync, with each page's local status.
- `sync` plans each output. **Without `--yes` it only shows the plan.** `--generate` runs the
  generators first; `--force` overwrites pages edited on the platform; `--prune` trashes pages
  whose file is gone.

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
