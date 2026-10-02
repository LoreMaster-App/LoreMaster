package platformport

import "fmt"

// PageNotFoundError means the page does not exist (any more) or is not visible.
type PageNotFoundError struct{ ID string }

func (e *PageNotFoundError) Error() string {
	return fmt.Sprintf("page %s does not exist or is not visible to this account", e.ID)
}

// VersionConflictError means someone changed the page after ExpectedVersion.
type VersionConflictError struct {
	ID              string
	ExpectedVersion int
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("page %s was edited after version %d; it was not overwritten", e.ID, e.ExpectedVersion)
}

// TitleTakenError means another page in the space has the title.
type TitleTakenError struct{ Title string }

func (e *TitleTakenError) Error() string {
	return fmt.Sprintf("another page in this space is already titled %q; give the document a \"title:\" override in its lore-master annotation", e.Title)
}
