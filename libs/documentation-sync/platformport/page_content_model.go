package platformport

// PageContent is a page's body read for a two-way sync pull: the neutral Document the page
// renders, the page's current version (so the pull can stamp the annotation), and flags naming
// anything in the platform's format that could not be converted faithfully. A non-empty Flags
// means the conversion was lossy and the caller should not overwrite the file.
type PageContent struct {
	Version int
	Body    Document
	Flags   []string
}
