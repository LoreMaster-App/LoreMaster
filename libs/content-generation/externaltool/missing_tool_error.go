package externaltool

import "fmt"

// MissingToolError says a tool a generator needs is not available, and how to get it.
type MissingToolError struct {
	// Tool is the command that was looked for.
	Tool string
	// Hint is what to run to install it, in the user's terms.
	Hint string
}

// Error words the problem and the remedy.
func (e *MissingToolError) Error() string {
	return fmt.Sprintf("%s was not found; %s", e.Tool, e.Hint)
}
