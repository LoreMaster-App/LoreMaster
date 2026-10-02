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
	// attachmentsChanged says a file the page shows differs from the one uploaded at
	// the last sync, which changes the page as much as an edit to the text does.
	attachmentsChanged bool
}

// detectChange decides an annotated page's action. A remote version newer than the
// one recorded at the last sync means someone edited the page: it is a conflict and
// nothing else is considered, so their edit is never overwritten. Otherwise every
// difference is collected, and the kind names the most significant: a body change is
// an update, a new parent a move, a new title a rename; none is unchanged.
func detectChange(remote remoteState, local localState) (ActionKind, []Change) {
	if remote.remoteVersion > remote.annotationVersion {
		return Conflict, nil
	}
	var changes []Change
	if local.contentHash != remote.annotationHash || local.attachmentsChanged {
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
