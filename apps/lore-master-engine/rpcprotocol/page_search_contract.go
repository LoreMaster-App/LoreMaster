package rpcprotocol

// MethodPageSearch finds pages in a space whose title contains the query, ignoring
// case, ordered by title; an empty query lists the space's pages. Result: PagesResult.
const MethodPageSearch = "page/search"

// PageSearchParams is the search.
type PageSearchParams struct {
	SessionID string `json:"sessionId"`
	SpaceKey  string `json:"spaceKey"`
	Query     string `json:"query"`
	// Limit caps the answer: 50 when absent, at most 200.
	Limit int `json:"limit,omitempty"`
}
