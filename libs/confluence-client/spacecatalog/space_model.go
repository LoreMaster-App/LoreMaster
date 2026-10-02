package spacecatalog

// Space is one Confluence space as the picker shows it.
type Space struct {
	// ID is the space id as a string (v2 sends a string, v1 a number).
	ID   string
	Key  string
	Name string
	// HomepageID is the space's home page, the default parent page for a sync; empty
	// when the space has none.
	HomepageID string
	// Personal is true for a user's personal space.
	Personal bool
}
