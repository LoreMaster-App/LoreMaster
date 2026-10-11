// Package editorsettings reads LoreMaster's configuration from the editor's own settings
// files (VS Code's settings.json), layered as the editor layers them: user, then the
// workspace folder, the most specific winning (#308). It is how a consumer that is not
// the editor, such as the MCP server or the CLI, sees the same configuration the
// extension does. It never reads a secret; credentials stay in the editor's secret store.
package editorsettings
