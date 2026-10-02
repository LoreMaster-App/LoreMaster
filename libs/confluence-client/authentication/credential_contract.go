package authentication

import "fmt"

// Credential is one of APIToken, PAT or Basic.
type Credential interface {
	// Kind is "apitoken", "pat" or "basic", as stored in settings.
	Kind() string
	fmt.Stringer
	fmt.GoStringer
}

// APIToken is an Atlassian account's email and API token: Confluence Cloud only.
type APIToken struct {
	Email string
	Token string
}

// PAT is a personal access token: Data Center and Server 7.9 or later.
type PAT struct {
	Token string
}

// Basic is a user name and password: Data Center and Server older than 7.9.
type Basic struct {
	User     string
	Password string
}

// Kind implements Credential.
func (APIToken) Kind() string { return "apitoken" }

// Kind implements Credential.
func (PAT) Kind() string { return "pat" }

// Kind implements Credential.
func (Basic) Kind() string { return "basic" }

// String names the credential and redacts its secret, so a credential that reaches a
// log or an error message never leaks it.
func (c APIToken) String() string {
	return fmt.Sprintf("APIToken{Email: %s, Token: %s}", c.Email, redacted(c.Token))
}

func (c PAT) String() string { return fmt.Sprintf("PAT{Token: %s}", redacted(c.Token)) }

func (c Basic) String() string {
	return fmt.Sprintf("Basic{User: %s, Password: %s}", c.User, redacted(c.Password))
}

// GoString redacts too, so %#v is as safe as %v.
func (c APIToken) GoString() string { return c.String() }

// GoString redacts too, so %#v is as safe as %v.
func (c PAT) GoString() string { return c.String() }

// GoString redacts too, so %#v is as safe as %v.
func (c Basic) GoString() string { return c.String() }

func redacted(secret string) string {
	if secret == "" {
		return "(empty)"
	}

	return "(redacted)"
}
