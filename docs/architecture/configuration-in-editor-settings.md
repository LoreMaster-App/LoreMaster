# ADR: Configuration lives in the editor's settings

Status: accepted (2026-10-11). Issue: #308. Supersedes the "Config file" decision
(`.lore-master.yaml`, #92) once the migration ships.

## Context

`.lore-master.yaml` is the only place the interesting options live, and opening it shows
bare keys: no explanation, no allowed values, no schema, no completion. The extension
already has a few `loreMaster.*` settings, so there are two configuration surfaces and the
richer one is the poorer to use. Values that belong to the person, not the project (the
Confluence base URL, the edition, the e-mail), have to be repeated in every workspace.

## Decision

The editor's settings are the source of truth. Every option is a contributed `loreMaster.*`
setting with a description, allowed values and a default; the structured ones (outputs,
generators) carry a JSON Schema so `settings.json` also gives hover, completion and
validation.

### Precedence

Standard VS Code layering, most specific wins:

**folder > workspace > user > default**

A base URL set once in User settings is inherited by every workspace and overridden only
where a workspace or folder says otherwise. `loreMaster.confluence.baseUrl` is a top-level
default; an output may override it.

### Secrets

Tokens and PATs never go in settings (plaintext, uploaded by Settings Sync, committed when
in workspace settings). They stay in the editor's secret store, keyed by base URL
(`loreMaster.credential.<baseUrl>`), which is per user and shared by every workspace.

### Who reads what

| Consumer | Source of configuration |
|---|---|
| VS Code extension | `workspace.getConfiguration`; hands the resolved settings to the engine per session |
| MCP server | VS Code configuration, workspace first, then user (folder over workspace) |
| GitHub Action | Action inputs only: one input per option, validated by the same schema; no settings file is read |
| CLI | Proposed: as the MCP server, plus flags and environment variables for CI (to confirm) |
| Other editor shells | Their own settings surface; the engine contract is the settings schema, not VS Code's API |

### Migration

On first activation, an existing `.lore-master.yaml` is imported into workspace settings and
the user is told once. The file stays a deprecated fallback for one minor version, with a
warning, and is then removed.

## Consequences

- A repository that gitignores `.vscode/` has no committed configuration for the CLI and
  MCP server; it passes it explicitly.
- The settings schema becomes the engine's public configuration contract and must be
  versioned like the RPC surface.
- Comment-preserving YAML merging (`yaml_merge_algorithm.go`) goes away with the file.
