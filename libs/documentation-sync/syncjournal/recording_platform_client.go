package syncjournal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"lore-master/libs/documentation-sync/platformport"
)

// Recorder wraps a platform and notes every page a sync creates, updates or trashes, taking
// a snapshot of a page before the first change so it can be put back. Reads and everything
// else pass through untouched.
type Recorder struct {
	platformport.DocumentationPlatform
	mu      sync.Mutex
	changes []PageChange
	index   map[string]int
}

var _ platformport.DocumentationPlatform = (*Recorder)(nil)

// NewRecorder wraps inner.
func NewRecorder(inner platformport.DocumentationPlatform) *Recorder {
	return &Recorder{DocumentationPlatform: inner, index: map[string]int{}}
}

// Changes returns what was recorded, in the order the pages were first changed.
func (r *Recorder) Changes() []PageChange {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]PageChange(nil), r.changes...)
}

// CreatePage implements platformport.DocumentationPlatform.
func (r *Recorder) CreatePage(ctx context.Context, page platformport.NewPage) (platformport.RemotePage, error) {
	created, err := r.DocumentationPlatform.CreatePage(ctx, page)
	if err != nil {
		return created, err
	}
	r.note(PageChange{Kind: Created, PageID: created.ID, Title: created.Title, AfterVersion: created.Version})

	return created, nil
}

// UpdatePage implements platformport.DocumentationPlatform. The page is snapshotted first; a
// change that could not be undone is not made.
func (r *Recorder) UpdatePage(ctx context.Context, update platformport.PageUpdate) (platformport.RemotePage, error) {
	var before *platformport.PageSnapshot
	if !r.seen(update.ID) {
		snapshot, err := r.SnapshotPage(ctx, update.ID)
		if err != nil {
			return platformport.RemotePage{}, fmt.Errorf("could not keep the page's current version for a rollback: %w", err)
		}
		before = &snapshot
	}
	updated, err := r.DocumentationPlatform.UpdatePage(ctx, update)
	if err != nil {
		return updated, err
	}
	r.note(PageChange{Kind: Updated, PageID: updated.ID, Title: titleBefore(before, updated.Title), Before: before, AfterVersion: updated.Version})

	return updated, nil
}

// TrashPage implements platformport.DocumentationPlatform.
func (r *Recorder) TrashPage(ctx context.Context, id string) error {
	var before *platformport.PageSnapshot
	snapshot, err := r.SnapshotPage(ctx, id)
	if err == nil {
		before = &snapshot
	} else {
		var missing *platformport.PageNotFoundError
		if !errors.As(err, &missing) {
			return fmt.Errorf("could not keep the page before trashing it: %w", err)
		}
	}
	if err := r.DocumentationPlatform.TrashPage(ctx, id); err != nil {
		return err
	}
	if before != nil {
		r.note(PageChange{Kind: Trashed, PageID: id, Title: before.Title, Before: before})
	}

	return nil
}

func titleBefore(before *platformport.PageSnapshot, fallback string) string {
	if before != nil {
		return before.Title
	}

	return fallback
}

func (r *Recorder) seen(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.index[id]

	return ok
}

// note records a change, or advances an earlier one for the same page: a page created and then
// updated in the same sync is still a created page, left at its latest version.
func (r *Recorder) note(change PageChange) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if at, ok := r.index[change.PageID]; ok {
		if change.Kind == Trashed {
			r.changes[at].Kind = Trashed
		}
		r.changes[at].AfterVersion = change.AfterVersion

		return
	}
	r.index[change.PageID] = len(r.changes)
	r.changes = append(r.changes, change)
}
