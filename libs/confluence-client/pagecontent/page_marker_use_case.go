package pagecontent

import (
	"context"
	"errors"
)

// The marker says "this page is Lore Master's". Orphan detection and prune consider
// only marked pages, so a page someone made by hand under the sync's root is never
// reported or trashed.
const (
	// MarkerLabel is the label every page Lore Master creates carries.
	MarkerLabel = "lore-master"
	// SourcePathProperty is the content property holding the workspace-relative path of
	// the Markdown file the page comes from.
	SourcePathProperty = "lore-master.source-path"
)

// sourcePathValue is the JSON value of SourcePathProperty.
type sourcePathValue struct {
	SourcePath string `json:"sourcePath"`
}

// MarkPage labels the page and records sourcePath in its content property, updating
// the property when the file has moved. Both steps are idempotent, so marking an
// already-marked page is safe.
func (p *Pages) MarkPage(ctx context.Context, id string, sourcePath string) error {
	if id == "" || sourcePath == "" {
		return errors.New("marking a page needs its id and the source file's path")
	}
	if err := p.api.addLabel(ctx, id, MarkerLabel); err != nil {
		return err
	}

	return p.api.setProperty(ctx, id, SourcePathProperty, sourcePathValue{SourcePath: sourcePath})
}
