# lore-master-jetbrains

The JetBrains shell (IntelliJ IDEA, WebStorm, PyCharm, GoLand, Rider, ...) over the same `lore-master-engine` sidecar the
VS Code extension uses. Epic: #433. Design: `docs/architecture/engine-and-shells.md`. User guide: `docs/jetbrains.md`.

Kotlin, Gradle (IntelliJ Platform Gradle Plugin 2.x), JDK 21, IntelliJ Platform 2025.1+. Files are `<snake>_<role>.kt`, one
package per slice under `src/main/kotlin/loremaster/`, tests beside them under `src/test/kotlin/`.

| Slice | What it does |
|---|---|
| `engineprocess` | Finds the engine binary for this platform, frames JSON-RPC over its stdio, matches responses to requests, answers the engine's own requests, owns the one engine process |
| `enginecheck` | Tools > LoreMaster: Check Engine |
| `engineprogress` | Runs engine work under a progress bar and reports in a balloon |
| `pluginsettings` | Settings > Tools > LoreMaster: the engine path, then the user settings the engine describes |
| `projectsettings` | Reads the project's outputs through the engine; opens `.vscode/settings.json` |
| `connection` | Connect to Confluence, credentials in the IDE's password safe, session open/close |
| `sync` | Sync to Confluence: plan, review dialog, execute with progress |
| `rollback` | Roll back a sync: journals, dry run, undo |
| `watch` | Watch mode: debouncing batcher, routing by the engine, silent sync |
| `configdescription` | Asks the engine what can be configured (`config/describe`): platforms, fields, sign-in, settings |
| `outputs` | Add an output for any platform the engine describes: its fields, sign-in, spaces and parent page browser, write `loreMaster.outputs` |
| `pages` | Check, build and publish GitHub Pages / wiki outputs |
| `generators` | Run generators |
| `syncview` | The Sync tool window: Local and each storage, with their pages and actions |
| `exportmarkdown` | Export Markdown As... (every format of `lore-master-engine export`) |

## Test

```bash
cd apps/lore-master-jetbrains
./gradlew test                                              # unit tests and a headless IDE
go build -o /tmp/lore-master-engine ../../apps/lore-master-engine
LORE_MASTER_ENGINE=/tmp/lore-master-engine ./gradlew test   # also drives the real engine: ping, sessions, a whole sync
                                                            # against a fake Confluence, pages, export
./gradlew runIde                                            # a sandbox IDE with the plugin
```

Without `LORE_MASTER_ENGINE` the real-engine tests do nothing (and pass); CI sets it.

## Release

One zip serves every platform, with the engine for each under `bin/<os>-<arch>/`:

```bash
VERSION=0.2.0 npx nx run lore-master-engine:build-all   # the six engines into dist/platforms/
./gradlew buildPlugin                                    # build/distributions/lore-master-jetbrains-<pluginVersion>.zip (75 MB)
```

`.github/workflows/jetbrains.yml` tests every change and, on `main`, releases a new `pluginVersion` (set in `gradle.properties`):
a GitHub Release `lore-master-jetbrains@<version>` with the zip, and the JetBrains Marketplace too when the
`JETBRAINS_MARKETPLACE_TOKEN` secret exists. Bump `pluginVersion` to release again.

## Not done

Markdown editing and preview (the IDE has its own Markdown plugin), and a page-level sync status in the editor. Never run by
hand in a live IDE: the dialogs, the tool window, the progress bar and watch mode's file listener are covered by headless
tests, or not at all (the listener).
