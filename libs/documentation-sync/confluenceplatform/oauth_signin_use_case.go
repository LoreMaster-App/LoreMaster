package confluenceplatform

import (
	"context"
	"errors"
	"net/http"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/oauthsignin"
)

// OAuthTokens is what an interactive OAuth sign-in yields. The editor stores them in its
// secret store; the access token opens a session, the refresh token renews it.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	// ExpiresIn is the access token's lifetime in seconds, as the provider reported it.
	ExpiresIn int
}

// OAuthSignInOptions configures an interactive Data Center OAuth 2.0 sign-in.
type OAuthSignInOptions struct {
	// BaseURL is the site's address; any page URL of the site is accepted.
	BaseURL string
	// ClientID identifies the admin's incoming OAuth 2.0 link.
	ClientID string
	Scope    string
	// OpenBrowser shows the authorize URL to the user; the editor supplies it.
	OpenBrowser func(authorizeURL string) error
	// HTTPClient and Logger default as in httptransport.
	HTTPClient *http.Client
}

// SignInOAuth runs the Data Center OAuth 2.0 sign-in (authorization code with PKCE) and
// returns the tokens, so the editor can store them and open a session with the access
// token. It persists nothing. The engine calls this rather than the confluence-client
// directly, so the dependency still points one way, through this adapter.
func SignInOAuth(ctx context.Context, options OAuthSignInOptions) (OAuthTokens, error) {
	baseURL, err := connection.NormalizeBaseURL(options.BaseURL)
	if err != nil {
		return OAuthTokens{}, &ConnectError{Failure: InvalidAddress, Err: err}
	}
	tokens, err := oauthsignin.SignIn(ctx, oauthsignin.SignInRequest{
		BaseURL: baseURL, ClientID: options.ClientID, Scope: options.Scope,
		HTTPClient: options.HTTPClient, OpenBrowser: options.OpenBrowser,
	})
	if err != nil {
		return OAuthTokens{}, err
	}

	return OAuthTokens{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, ExpiresIn: tokens.ExpiresIn}, nil
}

// RefreshOAuthOptions configures renewing an access token from a refresh token.
type RefreshOAuthOptions struct {
	BaseURL      string
	ClientID     string
	RefreshToken string
	HTTPClient   *http.Client
}

// RefreshOAuth renews the access token from the refresh token. A refresh token the
// provider rejects — revoked or expired — comes back as *ReauthRequired, so the caller can
// ask the user to sign in again rather than report a plain failure.
func RefreshOAuth(ctx context.Context, options RefreshOAuthOptions) (OAuthTokens, error) {
	baseURL, err := connection.NormalizeBaseURL(options.BaseURL)
	if err != nil {
		return OAuthTokens{}, &ConnectError{Failure: InvalidAddress, Err: err}
	}
	tokens, err := oauthsignin.Refresh(ctx, options.HTTPClient, baseURL, options.ClientID, options.RefreshToken)
	if err != nil {
		return OAuthTokens{}, reauthError(err)
	}

	return OAuthTokens{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, ExpiresIn: tokens.ExpiresIn}, nil
}

// ReauthRequired means an OAuth sign-in can no longer be renewed and must be redone.
type ReauthRequired struct {
	Err error
}

func (e *ReauthRequired) Error() string { return e.Err.Error() }

func (e *ReauthRequired) Unwrap() error { return e.Err }

// reauthError re-wraps the confluence-client's revoked-grant error as this package's, so
// callers match one type without importing the client's.
func reauthError(err error) error {
	var revoked *oauthsignin.ReauthRequired
	if errors.As(err, &revoked) {
		return &ReauthRequired{Err: err}
	}

	return err
}
