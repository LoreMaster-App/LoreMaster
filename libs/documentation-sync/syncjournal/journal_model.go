package syncjournal

import (
	"time"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// ChangeKind says what a sync did to a page.
type ChangeKind string

const (
	// Created pages are deleted again on a rollback.
	Created ChangeKind = "created"
	// Updated pages (content, title or parent) are put back as they were.
	Updated ChangeKind = "updated"
	// Trashed pages (a prune) cannot be undone here; the platform's trash restores them.
	Trashed ChangeKind = "trashed"
)

// PageChange is one page a sync changed.
type PageChange struct {
	Kind   ChangeKind `json:"kind"`
	PageID string     `json:"pageId"`
	Title  string     `json:"title"`
	// Before is the page as it was before the sync touched it; absent for a created page.
	Before *platformport.PageSnapshot `json:"before,omitempty"`
	// AfterVersion is the version the sync left the page at. A page whose version is another
	// number was edited since, and is not rolled back without force.
	AfterVersion int `json:"afterVersion"`
}

// FileChange is one local file a sync rewrote, or created.
type FileChange struct {
	// Path is workspace-relative and '/'-separated.
	Path string `json:"path"`
	// Existed is false when the sync created the file, so a rollback removes it.
	Existed bool `json:"existed"`
	// Before is the file's content before the sync; empty when it did not exist.
	Before []byte `json:"before,omitempty"`
	// AfterHash is the SHA-256 of the content the sync left; a file that differs was edited since.
	AfterHash string `json:"afterHash"`
}

// Journal is everything one sync changed.
type Journal struct {
	ID            string                   `json:"id"`
	StartedAt     time.Time                `json:"startedAt"`
	WorkspaceRoot string                   `json:"workspaceRoot"`
	OutputIndex   int                      `json:"outputIndex"`
	Output        workspacesettings.Output `json:"output"`
	Pages         []PageChange             `json:"pages"`
	Files         []FileChange             `json:"files"`
}

// Empty is true when the sync changed nothing, so there is nothing to keep.
func (journal Journal) Empty() bool {
	return len(journal.Pages) == 0 && len(journal.Files) == 0
}

// Summary counts what a rollback would do.
type Summary struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Trashed int `json:"trashed"`
	Files   int `json:"files"`
}

// Summarize counts the journal's changes by kind.
func (journal Journal) Summarize() Summary {
	summary := Summary{Files: len(journal.Files)}
	for _, change := range journal.Pages {
		switch change.Kind {
		case Created:
			summary.Created++
		case Updated:
			summary.Updated++
		case Trashed:
			summary.Trashed++
		}
	}

	return summary
}
