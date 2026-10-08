package rpcprotocol

// MethodWorkspaceTree reads the workspace and its .lore-master.yaml and returns the page
// tree one output would sync, each page with what the files alone say about it. It needs
// no connection and writes nothing: the editor's Pages view is built on it, and refines
// the remote half with a read-only sync/plan.
const MethodWorkspaceTree = "workspace/tree"

// WorkspaceTreeParams names the workspace and the output.
type WorkspaceTreeParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Output is the index of the output in the settings' outputs list.
	Output int `json:"output"`
}

// Local statuses a tree node can carry; see syncplanning.LocalStatus.
const (
	TreeStatusNew          = "new"
	TreeStatusSynced       = "synced"
	TreeStatusLocalChanges = "local-changes"
)

// WorkspaceTreeResult is the tree, parents first.
type WorkspaceTreeResult struct {
	// Nodes are in sync order: every parent comes before its children.
	Nodes []TreeNode `json:"nodes"`
	// Warnings are discovery, parse and nesting notes; Problems are files that could not
	// be read and errors that stop a tree from being built.
	Warnings []string `json:"warnings,omitempty"`
	Problems []string `json:"problems,omitempty"`
}

// TreeNode is one page.
type TreeNode struct {
	// Path is the workspace-relative file, '/'-separated.
	Path string `json:"path"`
	// Title is the page title from the file (its H1, or an annotation
	// title, else the file name); PageTitle is what the platform shows, with the
	// output's title prefix.
	Title     string `json:"title"`
	PageTitle string `json:"pageTitle"`
	// Parent is the file this page nests under; empty means directly under the
	// configured parent page.
	Parent string `json:"parent,omitempty"`
	// Rule is the nesting rule that placed the page.
	Rule  string `json:"rule"`
	Depth int    `json:"depth"`
	// Status is new, synced or local-changes for an output that tracks pages in the
	// files; empty for one that does not (a github-pages output).
	Status string `json:"status,omitempty"`
	// PageID is the platform page the file's annotation points at, when it has one.
	PageID   string   `json:"pageId,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
