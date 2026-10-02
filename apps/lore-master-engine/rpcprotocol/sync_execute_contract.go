package rpcprotocol

// MethodSyncExecute carries out a plan made by sync/plan, then writes the annotations
// back into the files. Progress arrives as host/progress notifications while it runs.
const MethodSyncExecute = "sync/execute"

// SyncExecuteParams names the plan and the user's choices.
type SyncExecuteParams struct {
	PlanID string `json:"planId"`
	// Force overwrites pages edited on the platform since the last sync.
	Force bool `json:"force,omitempty"`
	// Prune moves orphan pages to the trash.
	Prune bool `json:"prune,omitempty"`
}

// SyncExecuteResult is the report.
type SyncExecuteResult struct {
	Pages []PageOutcome `json:"pages"`
	// Rewritten are the files whose annotation changed.
	Rewritten []string `json:"rewritten,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

// PageOutcome is what happened to one page.
type PageOutcome struct {
	Path    string `json:"path,omitempty"`
	Title   string `json:"title"`
	Planned string `json:"planned"`
	// Outcome is written, unchanged, skipped, reported, trashed or failed.
	Outcome string `json:"outcome"`
	PageID  string `json:"pageId,omitempty"`
	Version int    `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
}
