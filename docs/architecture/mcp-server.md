# The MCP server

LoreMaster ships a [Model Context Protocol](https://modelcontextprotocol.io) server so an
AI agent that writes documentation knows how the sync will nest and title its files.
Without it the agent guesses: it picks the wrong parent, collides with an existing title or
misnames a dotted file, and the page lands in the wrong place on Confluence. Tracked in the
epic #211.

## Where it lives

In the engine, as a second mode of the same binary: `lore-master-engine --mcp`. It speaks
newline-delimited JSON-RPC 2.0 on stdio, which is both what the engine already does for the
editor shells and what the MCP stdio transport is. It is a small hand-written server in the
`mcpserver` slice (tools only; no resources or prompts yet) rather than a dependency, and
that choice can be revisited if the tool surface grows (#212).

```
VS Code ───registers───▶  lore-master-engine --mcp --workspace <folder>
external client ─config─▶        │
                                 ├─ mcpserver            protocol + the four tools
                                 ├─ markdown-workspace   discovery, parsing, the nesting rules
                                 └─ workspacesettings    .lore-master.yaml: roots, excludes, ignore
```

## One source of truth

The invariant: **what the MCP states is exactly what the sync enforces.** The tools do not
restate the rules; they call the code that applies them.

- The rules text comes from `documenttree.NestingConventions()`, which sits beside
  `NestingPolicy`, the code that places pages. Change one and the other is changed in the
  same file.
- `preview_tree`, `validate_document` and `place_document` load the workspace through the
  same discovery (`documentdiscovery`, driven by the `.lore-master.yaml` roots, excludes and
  ignore list) and build the tree with the same `documenttree.BuildTree` the sync uses.

So the advice cannot drift from the behaviour, and a `.lore-master.yaml` change shows up in
the tools without any MCP-side edit.

## The tools

| Tool | Input | Returns |
|---|---|---|
| `loremaster_nesting_rules` | none | The rules in the order they apply, as readable text and as structured `conventions` (`key`, `summary`): explicit `parent:`, dotted file name, directory index, selected parent, then titling (H1, `<titlePrefix>: <title>`, per-space uniqueness). |
| `preview_tree` | none | The pages the sync would create, as a tree: each page's file, title, parent and the rule that placed it, plus discovery and parse notes. |
| `validate_document` | `path` (required): workspace-relative path to a Markdown file | Where the file will nest and why, whether its title clashes with another page, whether it lacks an H1, and whether the sync includes it at all. |
| `place_document` | `h1` (required); `parent` (optional): a page title or workspace-relative file path; `directory` (optional): workspace-relative directory | The file path, and any annotation, that nests a new page under the chosen parent. It writes nothing; the agent creates the file. |

The three workspace-aware tools need the workspace root. Without one they return a tool
error saying so, and the rules tool still works.

A tool that fails for an operational reason returns a normal result with `isError` set, so
the model reads the explanation and can react; only a malformed call is a protocol error.

## How a client reaches it

- **In VS Code** the extension contributes an MCP server definition provider
  (`loreMaster.engine`) that launches the bundled engine with `--mcp --workspace <the open
  folder>` and re-advertises it when the workspace folders change (#215).
- **Elsewhere** the "LoreMaster: Copy MCP Server Config" command puts a ready-to-paste
  `mcpServers` entry on the clipboard, pointing at the same engine binary and workspace
  (#216).

Usage steps are in [Using the MCP server](../mcp-server.md).

## Decisions

- **Rules plus workspace-aware tools**, not rules alone: an agent that can see the tree and
  preview a placement makes fewer mistakes than one that only reads prose.
- **Auto-registration in the editor, copy-config outside it**: zero setup where the user is
  already inside LoreMaster, one paste elsewhere.
- **No new binary and no npm package**: the engine ships inside the extension already, and
  distribution stays the Marketplace only.
