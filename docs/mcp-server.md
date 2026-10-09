# Using the MCP server

The MCP server lets an AI agent ask LoreMaster how to name and place a Markdown file before
it creates one. How it is built is in [the architecture note](architecture/mcp-server.md).

## In VS Code

Nothing to set up. The LoreMaster extension registers the server itself, so the editor's
agent (for example Copilot Chat in agent mode) can see it:

1. Install the extension and open the folder you document.
2. Open the chat's tools list (or run **MCP: List Servers**). A server named **LoreMaster**
   is listed. Start it if the editor asks.
3. Ask the agent to write documentation. It can now call the six tools below.

The server reads the **first workspace folder**. It uses the engine bundled with the
extension, or the one named by the `loreMaster.engine.path` setting.

## From another MCP client

Claude Desktop, Claude Code and other clients take an `mcpServers` entry. With the
extension installed:

1. Open the folder you document in VS Code.
2. Run **LoreMaster: Copy MCP Server Config** from the Command Palette.
3. Paste the result into the client's MCP configuration.

The copied config looks like this (the paths are yours):

```json
{
  "mcpServers": {
    "loremaster": {
      "command": "<path to lore-master-engine>",
      "args": ["--mcp", "--workspace", "<path to your workspace>"]
    }
  }
}
```

Without VS Code, run the engine yourself: `lore-master-engine --mcp --workspace <folder>`.
`--workspace` defaults to the working directory the process starts in.

## The tools

### `loremaster_nesting_rules`

No input. Returns the rules the sync applies, in order, so the agent can follow them while
it writes: a `parent:` line in the file's `<!-- lore-master -->` annotation wins; then a dotted file name
(`readme.architecture.md` under `readme.md`); then the directory's `README.md` or
`index.md`; otherwise the page goes under the parent you picked for the sync. Titles are the
first `# H1` (or a `title:` line in the annotation), published as `<titlePrefix>: <title>`, and must be
unique within the Confluence space.

### `preview_tree`

No input. Returns the page tree the sync would build for this workspace right now, with the
rule that placed each page. Use it to see the current structure before adding to it.

### `validate_document`

Input: `path`, the workspace-relative path of a Markdown file, for example
`docs/architecture/engine.md`. Returns where the file will nest and why, and warns when its
title clashes with another page, when it has no H1, or when the sync does not include it at
all (outside the configured roots, or ignored).

### `place_document`

Input: `h1` (required), `parent` (optional: an existing page's title or file path) and
`directory` (optional: workspace-relative). Returns the file path and any annotation that
nests a new page under that parent. It **writes nothing**: the agent creates the file with
what it returns.

### `site_publishing_plan`

No input. Reads the repository's `.github/workflows` and `.lore-master.yaml` and says how the
repository deploys to GitHub Pages and which setting puts the docs site **beside** that deploy:
build into the app's artifact folder (an Actions-source deploy), publish into a `path` folder of
the branch (a branch push), or own the branch (nothing else deploys). Returns the
`.lore-master.yaml` output and the workflow step to use, and warns about what would go wrong: an
app deploy that deletes the docs folder, a branch publish without `path`, a workflow `paths:`
filter that will not redeploy on a docs change. It **writes nothing**.

### `preview_site`

Input: `output` (optional: the position of the github-pages output; needed only when there are
several). Renders the site in memory and returns how many pages and files it would contain, the
file list, and the Markdown problems that would stop a build. It **writes nothing**; the build
itself is `lore-master-engine pages build --out DIR` or the
[GitHub Action](github-action.md).

## Troubleshooting

- **The workspace tools say there is no workspace.** The server was started without
  `--workspace` and outside a folder. Pass `--workspace`, or start it from the folder.
- **A file you expect is missing from `preview_tree`.** It is outside the `roots`, matched by
  `excludes` or `ignore`, or ignored by a `.gitignore` while `skipGitignored` is on. See
  [Configuration](../README.md#configuration--lore-masteryaml).
- **The server is not listed in VS Code.** Update the extension and VS Code: MCP server
  registration needs a recent VS Code release.
