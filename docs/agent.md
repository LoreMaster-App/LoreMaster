# A documentation agent that follows the LoreMaster structure

An AI agent that writes documentation without knowing how LoreMaster nests and titles pages
guesses: the file lands in the wrong place, the title clashes with another page, a parent
that does not exist is named. **LoreMaster: Create documentation agent** writes an agent
definition into your workspace that knows the rules, so the pages it writes sync where you
expect.

## Create it

Command Palette → **LoreMaster: Create documentation agent**. Pick the tools that should get
the agent:

| Target | File | Used by |
|---|---|---|
| **Claude Code subagent** | `.claude/agents/loremaster-writer.md` | Claude Code, which delegates documentation work to it |
| **VS Code custom agent** | `.github/agents/loremaster-writer.agent.md` | the agent picker in VS Code's chat |
| **AGENTS.md block** | a delimited block in `AGENTS.md` | any agent that reads `AGENTS.md` |

The first two are preselected. The `AGENTS.md` block is added after what you already have,
and your own text is never changed.

## What it is told

The instructions are written by the engine each time, from the code that applies the rules,
so they cannot say anything the sync does not do:

- **The rules**, in the order the sync applies them: an explicit `parent:` line in the file's
  `<!-- lore-master -->` annotation, a dotted file name, the directory's `README.md`, the
  parent you selected for the sync; then how titles are formed, prefixed and kept unique.
- **Your workspace**: the storages and the title prefix, the folders that are synced and what
  is left out (`ignore`, excludes, `.gitignore`), whether sync is two-way, how Mermaid
  diagrams are handled, and which folders are **generated** and must not be edited by hand.
- **How to work**: look at the existing tree first, place the page, write it, validate it,
  leave the sync's own annotation keys alone (only `parent:` and `title:` are the author's),
  and leave running the sync to you.

It uses the LoreMaster MCP tools when it has them (`loremaster_nesting_rules`, `preview_tree`,
`place_document`, `validate_document`, and for the GitHub Pages site `site_publishing_plan` and
`preview_site`) and works from the written rules when it does not. In
VS Code the MCP server is registered for you; for Claude Code and other clients, **LoreMaster:
Copy MCP Server Config** gives you the entry (the command offers it when it finishes). See
[Using the MCP server](mcp-server.md).

## Keep it current

Run the command again after you change `.lore-master.yaml` (new folders, a new generator, a
new storage): the files it wrote are updated in place. A file it did not write — one you
created or edited by hand without its marker — is only replaced if you say so.
