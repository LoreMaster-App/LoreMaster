package platformport

import "context"

// DocumentationPlatform is everything the sync asks of a platform. Every method is
// safe to retry after an error except CreatePage, which an implementation makes
// idempotent itself when the platform allows (Confluence: by title, #24).
type DocumentationPlatform interface {
	ListSpaces(ctx context.Context) ([]Space, error)
	GetPage(ctx context.Context, id string) (RemotePage, error)
	// GetPageContent reads a page's body as a neutral Document for a two-way pull, with
	// flags naming anything in the platform's format that could not be converted faithfully.
	GetPageContent(ctx context.Context, id string) (PageContent, error)
	ListChildren(ctx context.Context, parentID string) ([]RemotePage, error)
	// FindPagesByTitle returns the pages in the space whose title equals title
	// ignoring case: the ones a new page of that title would clash with.
	FindPagesByTitle(ctx context.Context, space SpaceRef, title string) ([]RemotePage, error)
	// FindPages returns up to limit pages in the space whose title contains query,
	// ignoring case, ordered by title: what a page picker shows as someone types. An
	// empty query lists the space's pages.
	FindPages(ctx context.Context, space SpaceRef, query string, limit int) ([]RemotePage, error)
	CreatePage(ctx context.Context, page NewPage) (RemotePage, error)
	// UpdatePage writes a new version; a different ParentID moves the page.
	UpdatePage(ctx context.Context, update PageUpdate) (RemotePage, error)
	// MarkPage records that the page is the sync's, and which file it comes from.
	MarkPage(ctx context.Context, id string, sourcePath string) error
	// ListMarkedDescendants lists only pages MarkPage marked, at any depth below rootID.
	ListMarkedDescendants(ctx context.Context, rootID string) ([]RemotePage, error)
	// TrashPage moves the page to the trash; an already-gone page is not an error.
	TrashPage(ctx context.Context, id string) error
	UploadFile(ctx context.Context, pageID string, file File) (UploadedFile, error)
}
