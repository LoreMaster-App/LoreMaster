package syncplanning

import "slices"

// remoteState is what the planner knows about an annotated file's page.
type remoteState struct {
	annotationVersion int
	annotationHash    string
	remoteVersion     int
	remoteParentID    string
	remoteTitle       string
}

// localState is what the file wants the page to be.
type localState struct {
	contentHash string
	title       string
	// parentPageID is the parent the page should have; empty when that parent is
	// created in this sync, which always means a move.
	parentPageID string
	// renderedChanged says the page would show something else than at the last sync
	// although the file did not change: an attachment, or what a link resolves to.
	renderedChanged bool
}

// detectChange decides an annotated page's action by reconciling two signals: the
// remote changed when its version is newer than the one recorded at the last sync, and
// the file changed when its content hash differs (or what the page renders beyond the
// Markdown moved). The remote-changed case depends on direction:
//
//   - one-way (push): a remote change is always a conflict, left alone so the remote
//     edit is never overwritten;
//   - two-way: a remote change with the file unchanged is pulled back; with the file
//     also changed it is a conflict (no merge, by decision in two-way-sync.md).
//
// When the remote did not change, every difference is collected and the kind names the
// most significant: a body change is an update, a new parent a move, a new title a
// rename; none is unchanged — the same in both directions.
func detectChange(remote remoteState, local localState, twoWay bool) (ActionKind, []Change) {
	localContentChanged := local.contentHash != remote.annotationHash || local.renderedChanged
	if remote.remoteVersion > remote.annotationVersion {
		if twoWay && !localContentChanged {
			return Pull, []Change{ChangeContent}
		}

		return Conflict, nil
	}
	var changes []Change
	if localContentChanged {
		changes = append(changes, ChangeContent)
	}
	if local.parentPageID == "" || local.parentPageID != remote.remoteParentID {
		changes = append(changes, ChangeParent)
	}
	if local.title != remote.remoteTitle {
		changes = append(changes, ChangeTitle)
	}
	switch {
	case slices.Contains(changes, ChangeContent):
		return Update, changes
	case slices.Contains(changes, ChangeParent):
		return Move, changes
	case slices.Contains(changes, ChangeTitle):
		return RenameTitle, changes
	}

	return Unchanged, nil
}
