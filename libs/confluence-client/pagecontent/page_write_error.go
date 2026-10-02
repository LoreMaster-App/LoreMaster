package pagecontent

import "fmt"

// VersionConflictError means the page changed on Confluence since it was read:
// ExpectedVersion is the version the update was based on. The sync reports it as a
// conflict instead of overwriting someone's edit.
type VersionConflictError struct {
	ID              string
	ExpectedVersion int
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("Confluence page %s was edited after version %d; it was not overwritten", e.ID, e.ExpectedVersion)
}

// TitleTakenError means another page in the space already has the title.
type TitleTakenError struct {
	Title string
	// Message is Confluence's own explanation.
	Message string
}

func (e *TitleTakenError) Error() string {
	return fmt.Sprintf("another page in this space is already titled %q; give the document a \"title:\" override in its lore-master annotation", e.Title)
}
