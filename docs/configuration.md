# Configuration

LoreMaster is configured in your editor's settings, under `loreMaster.*`. Open them from
**Settings**, filtered with `@ext:LoreMaster.loremaster`, or from the command
**LoreMaster: Open settings**. Every option has a description, its allowed values and a
default.

Secrets are never settings. The Confluence token is asked for once and kept in the editor's
secret store, keyed by the Confluence address, so it serves every workspace.

## Where a value can live

Settings layer the way VS Code's always do, the most specific winning:

**folder > workspace > user > default**

Set what is yours once, in **User** settings, and let a workspace override it only where it
differs:

```jsonc
// User settings.json: applies to every workspace
{
  "loreMaster.confluence.baseUrl": "https://acme.atlassian.net/wiki",
  "loreMaster.defaults.mermaidMode": "image"
}
```

```jsonc
// .vscode/settings.json of one repository
{
  "loreMaster.outputs": [
    { "platform": "confluence", "space": "ENG", "parentPageId": "98306", "titlePrefix": "ENG" }
  ],
  "loreMaster.ignore": ["internal/"]
}
```

A list (`outputs`, `ignore`, `generators`) set at a more specific level **replaces** the
less specific one; it is not appended to it.

### Where the plugin saves

When you change a storage, a generator or the ignore list from the plugin, it writes to the
**`.code-workspace` file you have open** (its `settings`), and to the folder's
`.vscode/settings.json` only when there is no workspace file, so a workspace configured in its
`.code-workspace` file never grows a competing `.vscode` folder. The exception is a folder whose
`.vscode/settings.json` already holds LoreMaster keys: those win over the workspace file, so the
plugin keeps writing there. Comments and every other setting in the file are left as they were.

`space` is the space's key. On Confluence Cloud the address can use an alias that differs from
the key (`/wiki/spaces/LORE/...` for a space whose key is `TLM`), so LoreMaster accepts either, and
when it finds neither it lists the spaces the signed-in user can see.

## The settings

| Setting | What it is |
|---|---|
| `loreMaster.outputs` | The places the lore goes: `name`, platform, space, parent page, title prefix, direction, content, Mermaid mode, links, repository and branch, `include` and `exclude` lists, `template` ([page templates](templates.md)). |
| `loreMaster.confluence.baseUrl` | The Confluence address an output uses unless it sets its own `baseUrl`. |
| `loreMaster.defaults.*` | The `direction`, `mermaidMode`, `linkMode`, `titleCollision` and `titlePrefix` of an output that does not set them. All of them are in the Settings UI; a prefix set here means the first sync does not ask for one. |
| `loreMaster.ignore` | Gitignore-style patterns every output leaves out. |
| `loreMaster.skipGitignored` | Leave out Markdown that `.gitignore` ignores. Default on. |
| `loreMaster.generators` | Generators that write Markdown from test reports and source documentation. |

The editor's own preferences (`loreMaster.credentials.useKeychain`, `loreMaster.pages.*`, `loreMaster.generateBeforeSync`,
`loreMaster.watchDebounceSeconds`, `loreMaster.engine.path`, `loreMaster.export.*`) are separate and are not part of
the configuration.

A misspelt key or field is reported with the file and the key, and a key that looks like a
secret (`token`, `password`, ...) is refused.

## Who reads it

| | Reads |
|---|---|
| The VS Code extension, the engine, the MCP server | The settings above: the user's `settings.json`, then the folder's `.vscode/settings.json`, the folder winning. |
| The command line (`lore-master-engine sync`, `pages`, `generate`) | The same, plus the overrides below. Commit `.vscode/settings.json` for CI to find it. |
| The GitHub Action | The same, with its inputs (`outputs`, `ignore`, `skip-gitignored`, `generators`, `settings`) as the most specific layer. See [the action](github-action.md). |

### A multi-root workspace

A `.code-workspace` file's `settings` are the **workspace** layer, between your user settings and
the folder's. The extension hands the file's location to the engine itself. On the command line,
pass `--workspace-file team.code-workspace` (or set `LORE_MASTER_WORKSPACE_FILE`).

### Overrides for CI

`--settings` (or the `LORE_MASTER_SETTINGS` environment variable) takes the JSON of `loreMaster.*`
keys, or `@` and a file, and wins over every settings file:

```bash
lore-master-engine sync --yes --settings '{"loreMaster.outputs": [{"platform": "confluence", "space": "ENG", "parentPageId": "98306", "titlePrefix": "ENG"}]}'
```

The order, least to most specific: defaults, user settings, the `.code-workspace` file, the
folder's `.vscode/settings.json`, the overrides. A key that looks like a secret is refused here
too; credentials come from the environment variables in [the command line guide](cli.md#credentials).

## From `.lore-master.yaml`

The first time the extension activates in a workspace that has a `.lore-master.yaml` and no
`loreMaster.outputs` setting, it imports the file into `.vscode/settings.json` and tells you
once. The YAML is left where it is and is no longer read once the settings exist; delete it
when you like. A workspace with neither is configured in settings from the start.

The decision and its trade-offs are in
[architecture/configuration-in-editor-settings.md](architecture/configuration-in-editor-settings.md).

## Naming your storages

Give each output a `name` when you have more than one of a kind, such as one Confluence for
technical documentation and another for user manuals:

```jsonc
"loreMaster.outputs": [
  { "name": "Technical documentation", "platform": "confluence", "space": "ENG", "parentPageId": "98306" },
  { "name": "User manuals",            "platform": "confluence", "space": "MAN", "parentPageId": "55012" }
]
```

The name is what the sidebar tree, the "sync to…" picker, the rollback list and the CLI show
(`--output "User manuals"`), and it is the **Name** field of the storage form. Names are unique
(case-insensitive), at most 60 characters. A storage without one is shown with a name made from
the platform and where it goes (`Confluence · ENG`); the form fills that in when you save.
