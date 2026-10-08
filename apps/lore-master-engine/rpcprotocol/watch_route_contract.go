package rpcprotocol

// MethodWatchRoute says what a batch of changed files calls for: which generators read them
// and which Markdown pages they are, from the workspace's generators settings. The command line
// and the editors use it while watching, so only the affected generators run and only the
// affected pages sync. It reads the settings and nothing else.
const MethodWatchRoute = "watch/route"

// WatchRouteParams names the workspace and the files that changed.
type WatchRouteParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Changed are workspace-relative, '/'-separated paths.
	Changed []string `json:"changed"`
}

// WatchRouteResult is what to do about them.
type WatchRouteResult struct {
	// Everything is true when the settings file changed: run every generator, sync everything.
	Everything bool `json:"everything"`
	// Generators are indexes into the settings' generators list, ascending.
	Generators []int `json:"generators"`
	// Markdown are the changed Markdown files, sorted; sync these (with the pages the
	// generators write) rather than the whole workspace.
	Markdown []string `json:"markdown"`
}
