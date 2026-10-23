// Package notetags is what a note says: which tag it carries and the owner, issue and date written after it.
package notetags

// DefaultTags are the tags looked for when the settings name none.
var DefaultTags = []string{"TODO", "FIXME", "BUG", "HACK", "XXX", "NOTE", "WARN"}

// Note is one tagged comment.
type Note struct {
	// Path is workspace-relative and '/'-separated.
	Path string
	// Line and Column are 1-based; Column is where the tag starts.
	Line   int
	Column int
	// Tag is the tag as written, upper case.
	Tag string
	// Text is what follows the tag, without the owner, issue or date in brackets.
	Text string
	// Owner, Issue (without the #) and Due (YYYY-MM-DD) come from the brackets after the tag,
	// for example (alice), (#12), (2026-12-01) or (alice, #12).
	Owner string
	Issue string
	Due   string
}
