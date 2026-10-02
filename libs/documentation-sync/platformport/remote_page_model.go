package platformport

// Space is a place pages live in.
type Space struct {
	ID         string
	Key        string
	Name       string
	HomepageID string
}

// SpaceRef names a space; platforms differ in which of the two they need.
type SpaceRef struct {
	ID  string
	Key string
}

// RemotePage is the sync's view of a page on the platform.
type RemotePage struct {
	ID       string
	Title    string
	ParentID string
	// Version is the platform's version number; 0 when the listing did not say.
	Version int
	URL     string
}

// NewPage is a page to create.
type NewPage struct {
	Space    SpaceRef
	ParentID string
	Title    string
	Body     Document
}

// PageUpdate replaces a page's title, parent and body. ExpectedVersion is the version
// the change is based on; a newer remote version is a *VersionConflictError.
type PageUpdate struct {
	ID              string
	ExpectedVersion int
	Title           string
	ParentID        string
	Body            Document
	Message         string
}

// File is an attachment to upload.
type File struct {
	Name        string
	ContentType string
	Content     []byte
}

// UploadedFile is an attachment on a page.
type UploadedFile struct {
	ID   string
	Name string
	Hash string
	// Skipped is true when the page already had this content under this name.
	Skipped bool
}
