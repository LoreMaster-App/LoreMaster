package rpcprotocol

// MethodHostOpenExternal is a request from the engine to the editor: open a URL in the
// user's browser. The engine uses it to show the OAuth authorize page during an
// interactive sign-in; the editor answers once it has handed the URL to the browser.
const MethodHostOpenExternal = "host/openExternal"

// OpenExternalParams is the URL to open.
type OpenExternalParams struct {
	URL string `json:"url"`
}

// OpenExternalResult is the (empty) acknowledgement that the browser was opened.
type OpenExternalResult struct{}
