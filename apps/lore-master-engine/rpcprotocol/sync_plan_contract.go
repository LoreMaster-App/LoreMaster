package rpcprotocol

// MethodSyncPlan reads the workspace and its .lore-master.yaml and plans a sync of one
// output, writing nothing. The plan is kept in the engine under PlanID for
// sync/execute.
const MethodSyncPlan = "sync/plan"

// SyncPlanParams asks for a plan.
type SyncPlanParams struct {
	SessionID string `json:"sessionId"`
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Output is the index of the output in the settings' outputs list.
	Output int `json:"output"`
	// Scope limits the plan to these workspace-relative files ('/'-separated) and the
	// ancestors they need created; empty plans everything.
	Scope []string `json:"scope,omitempty"`
}

// SyncPlanResult is the plan. With Errors it cannot be executed; fix them and plan
// again.
type SyncPlanResult struct {
	PlanID   string         `json:"planId"`
	Actions  []PlanAction   `json:"actions"`
	Counts   map[string]int `json:"counts"`
	Warnings []string       `json:"warnings,omitempty"`
	Errors   []string       `json:"errors,omitempty"`
}

// PlanAction is what the sync will do to one page.
type PlanAction struct {
	// Kind is create, adopt, update, move, rename_title, unchanged, conflict or orphan.
	Kind string `json:"kind"`
	// Path is the workspace-relative file; empty for an orphan.
	Path  string `json:"path,omitempty"`
	Title string `json:"title"`
	// PageID and URL are set for a page that exists.
	PageID string `json:"pageId,omitempty"`
	URL    string `json:"url,omitempty"`
	// ParentPath is the file the page goes under; empty for the configured parent.
	ParentPath string `json:"parentPath,omitempty"`
	// Changes lists content, parent and title for update, move, rename and adopt.
	Changes []string `json:"changes,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}
