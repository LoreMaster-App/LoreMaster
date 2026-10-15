package rpcprotocol

import "time"

// MethodSyncJournals lists the syncs of a workspace that can still be rolled back, newest first.
const MethodSyncJournals = "sync/journals"

// MethodSyncRollback undoes a sync: the pages it created are deleted, the pages it updated are
// put back as they were and the local files get their earlier content back.
const MethodSyncRollback = "sync/rollback"

// SyncJournalsParams names the workspace.
type SyncJournalsParams struct {
	WorkspaceRoot string `json:"workspaceRoot"`
}

// JournalSummary is one rollbackable sync.
type JournalSummary struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"startedAt"`
	// Name is the storage's name, when it had one.
	Name string `json:"name,omitempty"`
	// BaseURL, Space and ParentPageID say where the sync wrote.
	BaseURL      string `json:"baseUrl"`
	Space        string `json:"space"`
	ParentPageID string `json:"parentPageId,omitempty"`
	Created      int    `json:"created"`
	Updated      int    `json:"updated"`
	Trashed      int    `json:"trashed"`
	Files        int    `json:"files"`
}

// SyncJournalsResult is the journals, newest first.
type SyncJournalsResult struct {
	Journals []JournalSummary `json:"journals"`
}

// SyncRollbackParams names the sync to undo and how.
type SyncRollbackParams struct {
	// SessionID is a session on the site the sync wrote to.
	SessionID     string `json:"sessionId"`
	WorkspaceRoot string `json:"workspaceRoot"`
	// JournalID is the sync to undo; empty means the latest.
	JournalID string `json:"journalId,omitempty"`
	// DryRun reports what would happen and changes nothing.
	DryRun bool `json:"dryRun,omitempty"`
	// Force also rolls back pages and files edited since the sync, discarding those edits.
	Force bool `json:"force,omitempty"`
}

// RollbackPage is what happened to one page.
type RollbackPage struct {
	PageID string `json:"pageId"`
	Title  string `json:"title"`
	// Kind is created, updated or trashed: what the sync did to it.
	Kind string `json:"kind"`
	// Result is trashed, restored, would-trash, would-restore, gone, skipped, failed or irreversible.
	Result string `json:"result"`
	Reason string `json:"reason,omitempty"`
}

// RollbackFile is what happened to one local file.
type RollbackFile struct {
	Path   string `json:"path"`
	Result string `json:"result"`
	Reason string `json:"reason,omitempty"`
}

// SyncRollbackResult is the report.
type SyncRollbackResult struct {
	JournalID string         `json:"journalId"`
	StartedAt time.Time      `json:"startedAt"`
	DryRun    bool           `json:"dryRun,omitempty"`
	Pages     []RollbackPage `json:"pages"`
	Files     []RollbackFile `json:"files"`
	// Complete is true when nothing was skipped or failed; the journal is then dropped.
	Complete bool `json:"complete"`
}
