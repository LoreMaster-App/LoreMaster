package rpcprotocol

// MethodPageSearch finds pages in a space whose title contains the query. Result:
// PagesResult.
const MethodPageSearch = "page/search"

// PageSearchParams is the search.
type PageSearchParams struct {
	SessionID string `json:"sessionId"`
	SpaceKey  string `json:"spaceKey"`
	Query     string `json:"query"`
}
