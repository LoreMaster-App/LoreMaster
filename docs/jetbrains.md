# LoreMaster for JetBrains IDEs

The same engine as the VS Code extension, in IntelliJ IDEA, WebStorm, PyCharm, GoLand, Rider and the other IntelliJ-based
IDEs (2025.1 or newer). The plugin is a thin shell: everything that touches Markdown or a platform runs in the bundled
`lore-master-engine`, so a sync, an export or a Pages build gives the same result in either editor.

## Install

From the JetBrains Marketplace (search for LoreMaster), or from disk: download `lore-master-jetbrains-<version>.zip` from the
[GitHub releases](https://github.com/LoreMaster-App/LoreMaster/releases) (tag `lore-master-jetbrains@<version>`), then
**Settings > Plugins > the gear > Install Plugin from Disk**. One zip serves every platform; it carries the engine for
Windows, Linux and macOS, on x64 and arm64.

## Configure

The configuration is the same `loreMaster.*` settings as in VS Code, in the project's **`.vscode/settings.json`**
(see [Configuration](configuration.md)). A team can use both editors on one repository. **Tools > LoreMaster: Open Project
Settings** opens the file, and creates it when it is missing:

```jsonc
{
  "loreMaster.outputs": [
    { "platform": "confluence", "baseUrl": "https://acme.atlassian.net/wiki", "space": "ENG", "parentPageId": "98306", "titlePrefix": "ENG" },
    { "platform": "github-pages", "repo": "acme/docs" }
  ]
}
```

The plugin's own preferences are in **Settings > Tools > LoreMaster**: the engine path (for a locally built engine), then the
settings the engine describes for the user (export folder, PDF page size, PNG width, browser, Word and OpenDocument templates,
watch mode's quiet time). That page, and the Add Output dialog, are drawn from the engine's `config/describe`, so a storage or a
setting the engine adds shows up without a new plugin version (see [Configuration](configuration.md#what-can-be-configured)).

## What it does

All of it is under **Tools**; the sync and export are also on the project view's context menu.

| Action | What it does |
|---|---|
| **LoreMaster: Connect to Confluence...** | Asks for the address and a way to sign in (Cloud email and API token, Data Center personal access token, or user and password), has the engine verify it, and keeps the credential in the IDE's **password safe**, one entry per site. Never in a file. |
| **LoreMaster: Sync...** | Plans the sync of an output, shows the plan (what is created, updated, moved, in conflict, orphaned), and on your yes runs it with the engine's progress in the IDE's progress bar. Select Markdown files or folders in the project view to sync only those. Asks for the credential once if there is none. |
| **LoreMaster: Add Output...** | Offers the storages the engine knows (Confluence, GitHub Pages, GitHub Wiki, and any it learns later), draws the fields of the one you choose, and adds the output to `loreMaster.outputs`. For Confluence it signs in, lists the spaces you can see and lets you browse or search for the parent page (double-click goes deeper); the title prefix starts as the parent page's title, so there are no page ids to find by hand. Less common settings are under *Show advanced settings*. |
| **LoreMaster: Roll Back a Sync...** | Lists the syncs that can still be undone, shows a dry run of what undoing the chosen one would do (pages to trash, pages to put back, local files to restore), and on your yes does it. If a page was edited since the sync it is left alone, unless you also choose to discard those edits. |
| **LoreMaster: Watch Mode** | A checked item. While on, saving a Markdown file, the settings or a file a generator reads syncs just what changed to your Confluence outputs after a quiet moment (Settings > Tools > LoreMaster > Watch mode), and runs only the generators that read it. It asks nothing: it never overwrites a page edited on the platform and never trashes, and it needs a saved sign-in (use Connect first). A failed sync is retried with a growing pause. |
| **LoreMaster: Check GitHub Pages** / **Build GitHub Pages Site...** / **Publish GitHub Pages** | What a publish would change, the static site into a folder you pick, or the site pushed to its branch with your own git. Works for `github-pages` and `github-wiki` outputs. |
| **LoreMaster: Run Generators** | Writes Markdown from test reports and source documentation. A generator's `prepare` shell command only runs if you say so. |
| **LoreMaster: Export Markdown As...** | On the editor, the project view and Tools: HTML, Word, OpenDocument, PDF, PNG, plain text, Jira wiki, Jira Cloud JSON and MediaWiki. Uses the editor's unsaved text. See [Export](export.md). |
| **LoreMaster: Check Engine** | Starts the engine and shows its version. |
| **LoreMaster: Forget Confluence Connection...** | Removes a site's credential from the password safe. |

## The Sync tool window

The **LoreMaster tool window** (right side) is the one place for your storages, laid out like the Sync view of the VS Code
extension. **Local** (every Markdown file in the repository) and each storage you have set up are rows, with their pages under
them as the tree the sync would make, new and changed pages marked. Double-click a page to open its file.

- **The toolbar on top** has what applies to every storage: **Add Sync Storage**, **LoreMaster Config**, **Refresh All Storages**,
  **Sync All Storages** and **Generate and Sync All**.
- **The toolbar under it, and a right-click on a row,** have the actions of the selected row (a tree cannot show icons on the
  row itself the way VS Code does). A storage: **Check Remote Status**, **Show File Names** (instead of page titles),
  **Flat View** (instead of a tree), **Hide Ignored** (the files it leaves out are listed under *Left out*), **Refresh Storage**,
  **Sync Storage**, **Generate and Sync**, **Edit Storage** and **Remove Storage**. Local has the display ones and Refresh,
  since it is not a place pages are sent to. The choices are shown as checked, and each row remembers its own.
- The view reads the files again when Markdown, the settings or the sync state change, so it follows your edits.

## Mermaid, PDF and PNG

The engine carries the Mermaid library, so diagrams are drawn in a sync, and in HTML, PDF and PNG exports, with nothing to
install. PDF and PNG are printed by a browser: install Chrome, Edge or Chromium (or set the path in the settings).

## What it does not do

Markdown editing and preview are the IDE's own Markdown support; the shortcuts, tables and table-of-contents tools of the
VS Code extension are not repeated here.

## Build it yourself

```bash
cd apps/lore-master-jetbrains
./gradlew test                                              # unit tests, and a headless IDE
LORE_MASTER_ENGINE=<a built engine> ./gradlew test          # also drives the real engine (sync, pages, export)
VERSION=0.2.0 npx nx run lore-master-engine:build-all       # the six engines
./gradlew buildPlugin                                       # build/distributions/lore-master-jetbrains-<pluginVersion>.zip
```

Releasing is a version bump: `pluginVersion` in `apps/lore-master-jetbrains/gradle.properties`. On `main` the
`JetBrains plugin` workflow tags `lore-master-jetbrains@<version>`, attaches the zip to a GitHub Release, and publishes to the
JetBrains Marketplace when the `JETBRAINS_MARKETPLACE_TOKEN` secret is set.
