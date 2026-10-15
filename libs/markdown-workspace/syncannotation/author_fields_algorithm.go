package syncannotation

import (
	"reflect"
	"slices"
)

// AuthorFields is the annotation with only what the author owns: the title and parent
// overrides, the generator marker and any key this version does not know. Everything the
// sync records (page id, version, hashes) is left out; it lives in lore-master.json.
func AuthorFields(annotation Annotation) Annotation {
	return Annotation{
		Title: annotation.Title, Parent: annotation.Parent, Generated: annotation.Generated,
		Unknown: slices.Clone(annotation.Unknown),
	}
}

// HasState reports whether the annotation records anything a sync wrote.
func HasState(annotation Annotation) bool {
	return annotation.PageID != "" || annotation.ParentID != "" || annotation.Version != 0 ||
		annotation.ContentHash != "" || annotation.RenderHash != "" || len(annotation.Attachments) > 0 ||
		!annotation.SyncedAt.IsZero()
}

// IsEmpty reports whether writing the annotation would put no field in the file.
func IsEmpty(annotation Annotation) bool {
	return reflect.DeepEqual(normalised(annotation), Annotation{})
}
