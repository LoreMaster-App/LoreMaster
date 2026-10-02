package syncplanning

import "lore-master/libs/markdown-workspace/documentdiscovery"

// SyncPlan is everything a sync would do, in an order the executor can follow: every
// page after its parent, orphans last. It serialises to JSON for the editor.
type SyncPlan struct {
	Actions []Action `json:"actions"`
	// Warnings are reported and do not stop the sync.
	Warnings []string `json:"warnings,omitempty"`
	// Errors stop the sync before anything is written.
	Errors []string `json:"errors,omitempty"`
}

// Action is what happens to one page.
type Action struct {
	Kind ActionKind `json:"kind"`
	// Path is the Markdown file; empty for an orphan.
	Path documentdiscovery.DocumentPath `json:"path,omitempty"`
	// Title is the page's final title (prefix included).
	Title string `json:"title"`
	// PageID is the existing page, empty for a create.
	PageID string `json:"pageId,omitempty"`
	// RemoteVersion is the existing page's current version, the one an update is
	// based on.
	RemoteVersion int `json:"remoteVersion,omitempty"`
	// ParentPath is the parent document; empty when the parent is the output's
	// configured parent page.
	ParentPath documentdiscovery.DocumentPath `json:"parentPath,omitempty"`
	// ParentPageID is the parent page when it already exists; empty when the parent is
	// created earlier in the same sync, in which case the executor uses its new id.
	ParentPageID string `json:"parentPageId,omitempty"`
	// Changes lists what an update, move or retitle changes.
	Changes []Change `json:"changes,omitempty"`
	// ContentHash is the file body's hash, written to the annotation after success.
	ContentHash string `json:"contentHash,omitempty"`
	// URL is the existing page's address, for the report.
	URL string `json:"url,omitempty"`
	// Reason explains a conflict, an orphan or a recreated page.
	Reason string `json:"reason,omitempty"`
}

// Counts summarises the plan by kind, for the editor's confirmation prompt.
func (p SyncPlan) Counts() map[ActionKind]int {
	counts := map[ActionKind]int{}
	for _, action := range p.Actions {
		counts[action.Kind]++
	}

	return counts
}
