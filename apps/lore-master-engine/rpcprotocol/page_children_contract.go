package rpcprotocol

// MethodPageChildren lists a page's direct children, for browsing to a parent page.
const MethodPageChildren = "page/children"

// PageChildrenParams names the page.
type PageChildrenParams struct {
	SessionID string `json:"sessionId"`
	PageID    string `json:"pageId"`
}

// PagesResult is a list of pages, already complete.
type PagesResult struct {
	Pages []Page `json:"pages"`
}

// Page is one page.
type Page struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parentId,omitempty"`
	URL      string `json:"url,omitempty"`
}
