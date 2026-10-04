package syncexecution

import (
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// SyncReport is the outcome of every planned action, in plan order.
type SyncReport struct {
	Pages    []PageResult `json:"pages"`
	Warnings []string     `json:"warnings,omitempty"`
}

// PageResult is one page's outcome.
type PageResult struct {
	Path    documentdiscovery.DocumentPath `json:"path,omitempty"`
	Title   string                         `json:"title"`
	Planned syncplanning.ActionKind        `json:"planned"`
	Outcome Outcome                        `json:"outcome"`
	PageID  string                         `json:"pageId,omitempty"`
	Version int                            `json:"version,omitempty"`
	URL     string                         `json:"url,omitempty"`
	Error   string                         `json:"error,omitempty"`
	// Annotation is what the file's annotation should say now; nil when the file
	// needs no rewrite (nothing was written, or the page is an orphan).
	Annotation *syncannotation.Annotation `json:"-"`
	// PulledBody is the new Markdown body to write below the annotation on a pull; nil
	// for every other outcome, where the file's body is left as it is.
	PulledBody []byte `json:"-"`
	// PulledAttachments are the image files a pull downloaded, to write into the workspace.
	PulledAttachments []PulledFile `json:"-"`
}

// PulledFile is one attachment a pull downloaded: its workspace-relative path and content.
type PulledFile struct {
	Path    documentdiscovery.DocumentPath
	Content []byte
}

// Count is the number of pages with the outcome.
func (r SyncReport) Count(outcome Outcome) int {
	count := 0
	for _, page := range r.Pages {
		if page.Outcome == outcome {
			count++
		}
	}

	return count
}
