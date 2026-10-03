package oauthsignin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Data Center serves the OAuth 2.0 provider at these paths, relative to the site base URL.
const (
	authorizePath = "/rest/oauth2/latest/authorize"
	tokenPath     = "/rest/oauth2/latest/token"
)

// PKCE is a code verifier and its S256 challenge (RFC 7636). The verifier is secret and
// sent only at the token exchange; the challenge travels in the authorize URL, so an
// intercepted code cannot be redeemed without the verifier.
type PKCE struct {
	Verifier  string
	Challenge string
}

// NewPKCE draws a verifier from random and derives its S256 challenge. random is
// crypto/rand.Reader in production; a fixed reader makes the pair deterministic in tests.
func NewPKCE(random io.Reader) (PKCE, error) {
	verifier, err := randomURLSafe(random, 32)
	if err != nil {
		return PKCE{}, err
	}
	sum := sha256.Sum256([]byte(verifier))

	return PKCE{Verifier: verifier, Challenge: base64.RawURLEncoding.EncodeToString(sum[:])}, nil
}

// AuthorizeRequest is what the browser needs to ask the user to approve the app.
type AuthorizeRequest struct {
	BaseURL     string
	ClientID    string
	RedirectURI string
	Scope       string
	State       string
	Challenge   string
}

// AuthorizeURL builds the Data Center authorization URL to open in the browser.
func AuthorizeURL(request AuthorizeRequest) string {
	query := url.Values{
		"client_id":             {request.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {request.RedirectURI},
		"state":                 {request.State},
		"code_challenge":        {request.Challenge},
		"code_challenge_method": {"S256"},
	}
	if request.Scope != "" {
		query.Set("scope", request.Scope)
	}

	return strings.TrimRight(request.BaseURL, "/") + authorizePath + "?" + query.Encode()
}

// Tokens is what the token endpoint returns. ExpiresIn is seconds; RefreshToken may be
// empty when the provider does not rotate it.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// ExchangeParams carries the authorization code back to be swapped for tokens.
type ExchangeParams struct {
	BaseURL     string
	ClientID    string
	Code        string
	RedirectURI string
	Verifier    string
}

// ExchangeCode swaps the authorization code for tokens (grant_type=authorization_code),
// proving possession of the PKCE verifier.
func ExchangeCode(ctx context.Context, httpClient *http.Client, params ExchangeParams) (Tokens, error) {
	return postToken(ctx, httpClient, params.BaseURL, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {params.Code},
		"redirect_uri":  {params.RedirectURI},
		"client_id":     {params.ClientID},
		"code_verifier": {params.Verifier},
	})
}

// Refresh gets a fresh access token from the refresh token. A provider that rejects the
// refresh token — the user revoked access, or it expired — yields *ReauthRequired.
func Refresh(ctx context.Context, httpClient *http.Client, baseURL string, clientID string, refreshToken string) (Tokens, error) {
	return postToken(ctx, httpClient, baseURL, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	})
}

// SignInRequest configures a full interactive sign-in.
type SignInRequest struct {
	BaseURL    string
	ClientID   string
	Scope      string
	HTTPClient *http.Client
	// OpenBrowser is handed the authorize URL to show the user; it is required (there is
	// no default way to open a browser from here — the editor shell supplies one).
	OpenBrowser func(authorizeURL string) error
	// Random defaults to crypto/rand.Reader.
	Random io.Reader
}

// SignIn runs the Data Center PKCE flow end to end: make a PKCE pair and a state, open the
// authorize URL in the browser, wait on a loopback port for the redirect, and exchange the
// code for tokens. The verifier is never logged or returned.
func SignIn(ctx context.Context, request SignInRequest) (Tokens, error) {
	if request.OpenBrowser == nil {
		return Tokens{}, errors.New("sign-in needs a way to open the browser")
	}
	random := request.Random
	if random == nil {
		random = rand.Reader
	}
	pkce, err := NewPKCE(random)
	if err != nil {
		return Tokens{}, err
	}
	state, err := randomURLSafe(random, 16)
	if err != nil {
		return Tokens{}, err
	}
	receiver, err := ListenLoopback()
	if err != nil {
		return Tokens{}, err
	}
	defer func() { _ = receiver.Close() }()

	authorizeURL := AuthorizeURL(AuthorizeRequest{
		BaseURL: request.BaseURL, ClientID: request.ClientID, RedirectURI: receiver.RedirectURI(),
		Scope: request.Scope, State: state, Challenge: pkce.Challenge,
	})
	if err := request.OpenBrowser(authorizeURL); err != nil {
		return Tokens{}, fmt.Errorf("could not open the browser for sign-in: %w", err)
	}
	code, err := receiver.Wait(ctx, state)
	if err != nil {
		return Tokens{}, err
	}

	return ExchangeCode(ctx, request.HTTPClient, ExchangeParams{
		BaseURL: request.BaseURL, ClientID: request.ClientID, Code: code,
		RedirectURI: receiver.RedirectURI(), Verifier: pkce.Verifier,
	})
}

func postToken(ctx context.Context, httpClient *http.Client, baseURL string, form url.Values) (Tokens, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	endpoint := strings.TrimRight(baseURL, "/") + tokenPath
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		return Tokens{}, fmt.Errorf("the token request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Tokens{}, tokenError(response.StatusCode, body)
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Tokens{}, fmt.Errorf("the token response was not valid JSON: %w", err)
	}
	if payload.AccessToken == "" {
		return Tokens{}, errors.New("the token response carried no access token")
	}

	return Tokens{AccessToken: payload.AccessToken, RefreshToken: payload.RefreshToken, ExpiresIn: payload.ExpiresIn}, nil
}

// tokenError words an OAuth error response. invalid_grant means the grant is dead — the
// code was used or the refresh token revoked — so it becomes *ReauthRequired; other
// failures stay generic. The body is never echoed whole, so a token in it cannot leak.
func tokenError(status int, body []byte) error {
	var payload struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &payload)
	if payload.Error == "invalid_grant" {
		return &ReauthRequired{Reason: cmpOr(payload.Description, payload.Error)}
	}
	detail := cmpOr(payload.Description, payload.Error)
	if detail == "" {
		detail = fmt.Sprintf("HTTP %d", status)
	}

	return fmt.Errorf("the token request was rejected (%s)", detail)
}

func randomURLSafe(random io.Reader, bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := io.ReadFull(random, buffer); err != nil {
		return "", fmt.Errorf("could not read randomness for the sign-in: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func cmpOr(value string, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}
