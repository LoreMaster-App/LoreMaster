package credentialverification

import "fmt"

// UnauthorizedError means the site refused the credential. Status is the HTTP status
// (401 or 403), or 200 when Cloud answered as an anonymous user.
type UnauthorizedError struct {
	Status int
}

func (e *UnauthorizedError) Error() string {
	switch e.Status {
	case 403:
		return "Confluence recognised the credential but refused access (HTTP 403); the account may lack the \"Can use\" permission, or the token may have expired"
	case 200:
		return "Confluence treated the request as anonymous; the credential was not accepted"
	}

	return fmt.Sprintf("Confluence rejected the credential (HTTP %d); check the email or user name and the token", e.Status)
}
