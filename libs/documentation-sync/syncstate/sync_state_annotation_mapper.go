package syncstate

import (
	"maps"
	"time"

	"lore-master/libs/markdown-workspace/syncannotation"
)

// PageFrom is the state an annotation carries.
func PageFrom(annotation syncannotation.Annotation) Page {
	page := Page{
		PageID: annotation.PageID, ParentID: annotation.ParentID, Version: annotation.Version,
		ContentHash: annotation.ContentHash, RenderHash: annotation.RenderHash,
		Attachments: maps.Clone(annotation.Attachments),
	}
	if len(page.Attachments) == 0 {
		page.Attachments = nil
	}
	if !annotation.SyncedAt.IsZero() {
		page.SyncedAt = annotation.SyncedAt.UTC().Format(time.RFC3339)
	}

	return page
}

// AnnotationFor is the annotation a file has once the state is applied: what the author set
// in the file, plus the recorded state of its page in storage.
func AnnotationFor(author *syncannotation.Annotation, storage Storage, page Page) *syncannotation.Annotation {
	var annotation syncannotation.Annotation
	if author != nil {
		annotation = syncannotation.AuthorFields(*author)
	}
	annotation.Platform, annotation.BaseURL, annotation.Space = storage.Platform, storage.BaseURL, storage.Space
	annotation.PageID, annotation.ParentID, annotation.Version = page.PageID, page.ParentID, page.Version
	annotation.ContentHash, annotation.RenderHash = page.ContentHash, page.RenderHash
	annotation.Attachments = maps.Clone(page.Attachments)
	if syncedAt, err := time.Parse(time.RFC3339, page.SyncedAt); err == nil {
		annotation.SyncedAt = syncedAt
	}

	return &annotation
}
