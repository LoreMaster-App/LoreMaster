package syncplanning

// ActionKind is what the sync will do to one page.
type ActionKind string

// Action kinds.
const (
	// Create makes a new page for a file that has none.
	Create ActionKind = "create"
	// Adopt takes over an existing page whose title matches an unannotated file
	// (titleCollision: adopt), then writes it like an update.
	Adopt ActionKind = "adopt"
	// Update writes a changed body (and any parent or title change with it).
	Update ActionKind = "update"
	// Move gives the page a new parent (and any title change with it).
	Move ActionKind = "move"
	// RenameTitle changes only the page title.
	RenameTitle ActionKind = "rename_title"
	// Unchanged needs nothing.
	Unchanged ActionKind = "unchanged"
	// Conflict means the page was edited on the platform since the last sync; it is
	// left alone and reported.
	Conflict ActionKind = "conflict"
	// Orphan is a page the sync made whose file no longer exists; prune trashes it.
	Orphan ActionKind = "orphan"
)

// Change is one aspect of a page that differs from its file.
type Change string

// Changes an update carries.
const (
	ChangeContent Change = "content"
	ChangeParent  Change = "parent"
	ChangeTitle   Change = "title"
)
