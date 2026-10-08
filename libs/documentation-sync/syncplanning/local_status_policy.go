package syncplanning

import (
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// LocalStatus is what a file alone says about its page, with no connection: it never
// reflects the platform, so a page edited there still reads as synced here.
type LocalStatus string

// Local statuses.
const (
	// LocalNew means no page of this output is recorded for the file: it has no
	// annotation, or the annotation belongs to another site or space.
	LocalNew LocalStatus = "new"
	// LocalSynced means the file is the one last synced.
	LocalSynced LocalStatus = "synced"
	// LocalChanged means the file was edited since the last sync.
	LocalChanged LocalStatus = "local-changes"
)

// LocalStatusOf classifies a document against its annotation, using the same "is this the
// output's page" rule and the same content hash the planner uses, so the offline status
// and a real plan agree about what is new and what changed locally.
func LocalStatusOf(document documentparsing.MarkdownDocument, output workspacesettings.Output) LocalStatus {
	annotation := document.Annotation
	if annotation == nil || annotation.PageID == "" || !sameTarget(annotation, output) {
		return LocalNew
	}
	if syncannotation.ContentHash(document.Body) != annotation.ContentHash {
		return LocalChanged
	}

	return LocalSynced
}
