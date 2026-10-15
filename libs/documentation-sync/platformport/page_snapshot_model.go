package platformport

// PageSnapshot is a page exactly as the platform held it at one moment: enough to put it
// back, so a sync can be rolled back. Markup is the platform's own body format and means
// nothing outside the platform that produced it.
type PageSnapshot struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parentId,omitempty"`
	Version  int    `json:"version"`
	Markup   string `json:"markup"`
}
