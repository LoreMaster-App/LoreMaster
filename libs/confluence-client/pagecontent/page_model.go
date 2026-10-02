package pagecontent

// Page is one Confluence page as the sync needs it.
type Page struct {
	ID    string
	Title string
	// SpaceID is always set on Cloud; SpaceKey on Data Center and Server (and in search
	// results everywhere). Both are set when the API returns both.
	SpaceID  string
	SpaceKey string
	// ParentID is the parent page, empty for a space's top-level page.
	ParentID string
	// Version is the page's current version number; zero when the API did not say
	// (Cloud's children listing omits it).
	Version int
	// WebURL is the absolute address of the page in the browser.
	WebURL string
	// BodyStorage is the page body in storage format, filled only when asked for.
	BodyStorage string
}
