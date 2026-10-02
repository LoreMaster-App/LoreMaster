package confluenceplatform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"lore-master/libs/confluence-client/authentication"
	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/credentialverification"
	"lore-master/libs/confluence-client/editiondetection"
	"lore-master/libs/confluence-client/httptransport"
	"lore-master/libs/confluence-client/storageformat"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// ConnectOptions is what a person types to connect to a Confluence site.
type ConnectOptions struct {
	// BaseURL is the site's address; any page URL of the site is accepted.
	BaseURL string
	// Edition is "cloud", "datacenter" or "server"; empty detects it.
	Edition string
	SignIn  SignIn
	// HTTPClient and Logger default as in httptransport.
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// SignIn is a credential as an editor collects it. Kind decides which fields are
// read: "apitoken" reads Email and Token, "pat" reads Token, "basic" reads User and
// Password.
type SignIn struct {
	Kind     string
	Email    string
	Token    string
	User     string
	Password string
}

// Connected is a verified connection.
type Connected struct {
	Platform *Platform
	// BaseURL is normalised; Edition is "cloud", "datacenter" or "server"; Version is
	// the server version ("8.5.4") on Data Center and Server, empty on Cloud.
	BaseURL, Edition, Version string
	// DisplayName is who signed in; AccountID is set on Cloud, Username elsewhere.
	DisplayName, AccountID, Username string
}

// ConnectFailure says why Connect failed, so a caller can tell the person what to do
// without knowing Confluence.
type ConnectFailure string

// Connect failures.
const (
	// InvalidAddress: the base URL is not a usable address.
	InvalidAddress ConnectFailure = "invalid_address"
	// InvalidCredential: the credential is incomplete or of an unknown kind.
	InvalidCredential ConnectFailure = "invalid_credential"
	// UnsupportedCredential: the edition does not take this kind of credential.
	UnsupportedCredential ConnectFailure = "unsupported_credential"
	// Unauthorized: the site refused the credential.
	Unauthorized ConnectFailure = "unauthorized"
	// Unreachable: the site did not answer, or is not a Confluence site.
	Unreachable ConnectFailure = "unreachable"
)

// ConnectError is a failed Connect. Its message names the remedy and never the secret.
type ConnectError struct {
	Failure ConnectFailure
	Err     error
}

func (e *ConnectError) Error() string { return e.Err.Error() }

func (e *ConnectError) Unwrap() error { return e.Err }

// Connect opens a verified connection to a Confluence site: the address is
// normalised, the edition detected when not given, the credential checked against
// what the edition accepts, then verified by asking who it belongs to.
func Connect(ctx context.Context, options ConnectOptions) (Connected, error) {
	baseURL, err := connection.NormalizeBaseURL(options.BaseURL)
	if err != nil {
		return Connected{}, &ConnectError{Failure: InvalidAddress, Err: err}
	}
	credential, err := credentialFrom(options.SignIn)
	if err != nil {
		return Connected{}, &ConnectError{Failure: InvalidCredential, Err: err}
	}

	site := connection.Connection{BaseURL: baseURL}
	if options.Edition != "" {
		if site.Edition, err = connection.ParseEdition(options.Edition); err != nil {
			return Connected{}, &ConnectError{Failure: InvalidAddress, Err: err}
		}
	} else {
		anonymous, err := httptransport.New(httptransport.Options{BaseURL: baseURL, HTTPClient: options.HTTPClient, Logger: options.Logger})
		if err != nil {
			return Connected{}, &ConnectError{Failure: InvalidAddress, Err: err}
		}
		detected, err := editiondetection.DetectEdition(ctx, anonymous, baseURL)
		if err != nil {
			return Connected{}, &ConnectError{Failure: Unreachable, Err: err}
		}
		site.Edition, site.Version = detected.Edition, detected.Version
	}

	platform, err := New(Options{Connection: site, Credential: credential, HTTPClient: options.HTTPClient, Logger: options.Logger})
	if err != nil {
		return Connected{}, &ConnectError{Failure: UnsupportedCredential, Err: err}
	}
	user, err := credentialverification.VerifyCredentials(ctx, platform.client)
	var refused *credentialverification.UnauthorizedError
	switch {
	case errors.As(err, &refused):
		return Connected{}, &ConnectError{Failure: Unauthorized, Err: err}
	case err != nil:
		return Connected{}, &ConnectError{Failure: Unreachable, Err: err}
	}

	connected := Connected{
		Platform: platform, BaseURL: baseURL, Edition: string(site.Edition),
		DisplayName: user.DisplayName, AccountID: user.AccountID, Username: user.Username,
	}
	if site.Version.Known() {
		connected.Version = site.Version.String()
	}

	return connected, nil
}

func credentialFrom(signIn SignIn) (authentication.Credential, error) {
	missing := func(fields ...string) error {
		return fmt.Errorf("a %s credential needs %v", signIn.Kind, fields)
	}
	switch signIn.Kind {
	case "apitoken":
		if signIn.Email == "" || signIn.Token == "" {
			return nil, missing("email", "token")
		}

		return authentication.APIToken{Email: signIn.Email, Token: signIn.Token}, nil
	case "pat":
		if signIn.Token == "" {
			return nil, missing("token")
		}

		return authentication.PAT{Token: signIn.Token}, nil
	case "basic":
		if signIn.User == "" || signIn.Password == "" {
			return nil, missing("user", "password")
		}

		return authentication.Basic{User: signIn.User, Password: signIn.Password}, nil
	}

	return nil, fmt.Errorf("the credential kind %q is not one of apitoken, pat or basic", signIn.Kind)
}

// SameSite reports whether two addresses name the same Confluence site once
// normalised, so a settings file's baseUrl matches the session that was opened for it
// whichever way the person wrote it.
func SameSite(a string, b string) bool {
	normalA, errA := connection.NormalizeBaseURL(a)
	normalB, errB := connection.NormalizeBaseURL(b)

	return errA == nil && errB == nil && strings.EqualFold(normalA, normalB)
}

// ForOutput is the same connection rendering pages the way the output asks: its
// mermaidMode and linkMode. Pages written through it use those; the original is
// unchanged.
func (p *Platform) ForOutput(output workspacesettings.Output) *Platform {
	copied := *p
	copied.render = storageformat.Options{LinkMode: storageformat.LinkByTitle, MermaidMode: storageformat.MermaidImage}
	if output.LinkMode == "id" {
		copied.render.LinkMode = storageformat.LinkByURL
	}
	switch output.MermaidMode {
	case "code":
		copied.render.MermaidMode = storageformat.MermaidCode
	case "html-macro":
		copied.render.MermaidMode = storageformat.MermaidHTMLMacro
	case "marketplace-macro":
		copied.render.MermaidMode = storageformat.MermaidMarketplaceMacro
	}

	return &copied
}
