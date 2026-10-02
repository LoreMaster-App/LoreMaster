package pagecontent

import "fmt"

// PageNotFoundError means the page does not exist (any more), or the user cannot see
// it: Confluence answers 404 for both.
type PageNotFoundError struct {
	ID string
}

func (e *PageNotFoundError) Error() string {
	return fmt.Sprintf("Confluence page %s does not exist or is not visible to this account", e.ID)
}
