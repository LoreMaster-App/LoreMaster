# Watch mode

Watch mode keeps the storage up to date while you work. It watches the Markdown, the generators'
inputs (source files, test reports, API descriptions) and `.lore-master.yaml`. When files change
and then stay quiet for a moment, it **regenerates only the generators that read what changed**
and **syncs only the pages that changed or were rewritten**, a scoped sync rather than a full one.

Edit a doc comment in `libs/core/store.go` and, a few seconds later, the `go-docs` generator
rewrites that package's page and the page alone is synced.

## From the command line

```bash
lore-master-engine watch --yes            # apply every batch
lore-master-engine watch                  # only plan each batch and show it
lore-master-engine watch --debounce 5s --poll 2s --output 0
```

| Flag | Meaning |
|---|---|
| `--yes` | Apply each batch. Without it the batch is generated and planned, and nothing is changed on the platform. |
| `--force` | Overwrite pages edited on the platform since the last sync. |
| `--debounce` | How long nothing may change before the changes are applied (default `2s`). Two quick edits are one sync. |
| `--poll` | How often the workspace is looked at (default `1s`). |
| `--output N` | Watch only the output at that position in the outputs list. |

Credentials come from the environment, as for `sync` (see [the command line](cli.md)). It runs
until interrupted with Ctrl+C.

## From the editor

Run **LoreMaster: Toggle watch mode** from the command palette, or the eye button in the
Generators view. The status bar shows `LoreMaster: watching`, `syncing`, or a warning when a sync
failed and is being retried; the **LoreMaster** output channel logs what each batch did. Nothing
is asked: plans run as planned, and a page edited on the platform is left alone rather than
overwritten. The setting `loreMaster.watchDebounceSeconds` (default 2) sets the quiet time.

Watch mode needs a storage already set up, so run **Sync** once first.

## How it behaves

- **Only what changed.** The engine's `watch/route` method maps each changed file to the
  generators that read it (by type and by the generator's `input` patterns) and to the Markdown
  pages. A generator's own output never re-triggers it.
- **Settings.** Changing `.lore-master.yaml` runs every generator and syncs everything.
- **Bursts.** Changes are gathered until nothing has changed for the debounce time. Changes made
  while a sync runs wait for the next one; two syncs never overlap.
- **Our own writes.** The pages a generator writes and the annotations a sync adds to your files
  are not taken for your edits, so a sync does not trigger the next one.
- **Failures.** A sync that fails is tried again with the same files (and any that changed since),
  after a wait that doubles from the debounce time up to a minute. A generator that fails is
  reported, and the rest of the batch still syncs. Plan errors and pages edited on the platform
  stop the batch (the command line says so) until the files change again.
- **Ignored.** Hidden folders, `node_modules`, `vendor`, `bin`, `obj`, `dist`, `build`, `out`,
  `target`, virtual environments and `testdata` are not watched.
- **GitHub Pages.** A GitHub Pages output publishes the whole site each batch, since it cannot be
  scoped.
