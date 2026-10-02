package httptransport

import "fmt"

// maxErrorBody bounds how much of a response body an APIError keeps.
const maxErrorBody = 2048

// APIError is a response Confluence answered with a status the transport does not
// retry, or kept answering after the last retry. Body is Confluence's message, cut to
// a readable length; it never contains the request.
type APIError struct {
	Method string
	Path   string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("%s %s: HTTP %d", e.Method, e.Path, e.Status)
	}

	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.Status, e.Body)
}
