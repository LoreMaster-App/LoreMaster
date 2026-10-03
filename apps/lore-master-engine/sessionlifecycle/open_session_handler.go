package sessionlifecycle

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"lore-master/apps/lore-master-engine/hostbridge"
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/confluenceplatform"
)

// Environment is how sessions reach the network.
type Environment struct {
	// HTTPClient defaults to the platform client's own.
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// OpenSession handles session/open: connect, verify, keep.
func OpenSession(store *Store, environment Environment) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SessionOpenParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		credential := params.Credential

		// An "oauth" credential with no access token but a client id starts an interactive
		// browser sign-in: the engine drives it, opening the browser through the editor, and
		// returns the tokens so the editor stores them. A credential that already carries an
		// access token (a later session) skips this and connects straight away.
		var issued *rpcprotocol.SessionTokens
		if credential.Kind == "oauth" && credential.AccessToken == "" && credential.ClientID != "" {
			tokens, err := confluenceplatform.SignInOAuth(ctx, confluenceplatform.OAuthSignInOptions{
				BaseURL: params.BaseURL, ClientID: credential.ClientID, Scope: credential.Scope,
				OpenBrowser: hostbridge.OpenBrowser(ctx, call.Editor, 0),
				HTTPClient:  environment.HTTPClient,
			})
			if err != nil {
				return nil, signInError(err)
			}
			credential.AccessToken = tokens.AccessToken
			issued = &rpcprotocol.SessionTokens{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, ExpiresIn: tokens.ExpiresIn}
		}

		connect := func(cred rpcprotocol.Credential) (confluenceplatform.Connected, error) {
			return confluenceplatform.Connect(ctx, confluenceplatform.ConnectOptions{
				BaseURL: params.BaseURL, Edition: params.Edition,
				SignIn: confluenceplatform.SignIn{
					Kind: cred.Kind, Email: cred.Email, Token: cred.Token,
					User: cred.User, Password: cred.Password, AccessToken: cred.AccessToken,
				},
				HTTPClient: environment.HTTPClient, Logger: environment.Logger,
			})
		}

		connected, err := connect(credential)
		// A stored OAuth access token that the site refuses is renewed from the refresh token
		// and tried once more; the fresh tokens go back in the result so the editor replaces
		// what it stored. A refresh token the provider has revoked is a re-auth, not a retry.
		if err != nil && credential.Kind == "oauth" && credential.RefreshToken != "" && credential.ClientID != "" && unauthorized(err) {
			tokens, refreshErr := confluenceplatform.RefreshOAuth(ctx, confluenceplatform.RefreshOAuthOptions{
				BaseURL: params.BaseURL, ClientID: credential.ClientID, RefreshToken: credential.RefreshToken, HTTPClient: environment.HTTPClient,
			})
			if refreshErr != nil {
				return nil, refreshError(refreshErr)
			}
			credential.AccessToken = tokens.AccessToken
			issued = &rpcprotocol.SessionTokens{AccessToken: tokens.AccessToken, RefreshToken: keepToken(tokens.RefreshToken, credential.RefreshToken), ExpiresIn: tokens.ExpiresIn}
			connected, err = connect(credential)
		}
		if err != nil {
			return nil, connectError(err)
		}
		session := store.Add(Session{BaseURL: connected.BaseURL, Edition: connected.Edition, Platform: confluence{connected.Platform}})

		return rpcprotocol.SessionOpenResult{
			SessionID: session.ID, BaseURL: connected.BaseURL, Edition: connected.Edition, Version: connected.Version,
			User:   rpcprotocol.SessionUser{DisplayName: connected.DisplayName, AccountID: connected.AccountID, Username: connected.Username},
			Tokens: issued,
		}, nil
	}
}

// signInError words a failed interactive OAuth sign-in for the editor. A bad address keeps
// its invalid-params code; anything else (the user cancelled, the provider refused, the
// browser could not open) is reported as a sign-in failure for the editor to show.
func signInError(err error) error {
	var failed *confluenceplatform.ConnectError
	if errors.As(err, &failed) {
		return connectError(err)
	}

	return rpcprotocol.Errorf(rpcprotocol.CodeUnauthorized, "the OAuth sign-in did not complete: %s", err.Error())
}

// unauthorized reports whether Connect failed because the site refused the credential.
func unauthorized(err error) bool {
	var failed *confluenceplatform.ConnectError

	return errors.As(err, &failed) && failed.Failure == confluenceplatform.Unauthorized
}

// refreshError words a failed token refresh. A revoked or expired refresh token is a
// re-auth (the editor prompts a fresh sign-in); anything else keeps a connect code.
func refreshError(err error) error {
	var reauth *confluenceplatform.ReauthRequired
	if errors.As(err, &reauth) {
		return rpcprotocol.Errorf(rpcprotocol.CodeReauthRequired, "the Confluence sign-in expired; sign in again (%s)", err.Error())
	}
	var failed *confluenceplatform.ConnectError
	if errors.As(err, &failed) {
		return connectError(err)
	}

	return rpcprotocol.Errorf(rpcprotocol.CodeUnauthorized, "could not refresh the Confluence sign-in: %s", err.Error())
}

// keepToken keeps the old refresh token when the provider did not issue a new one (not all
// rotate the refresh token on renewal).
func keepToken(fresh string, previous string) string {
	if fresh != "" {
		return fresh
	}

	return previous
}

func connectError(err error) error {
	var failed *confluenceplatform.ConnectError
	if !errors.As(err, &failed) {
		return err
	}
	codes := map[confluenceplatform.ConnectFailure]int64{
		confluenceplatform.InvalidAddress:        rpcprotocol.CodeInvalidParams,
		confluenceplatform.InvalidCredential:     rpcprotocol.CodeInvalidParams,
		confluenceplatform.UnsupportedCredential: rpcprotocol.CodeUnsupported,
		confluenceplatform.Unauthorized:          rpcprotocol.CodeUnauthorized,
		confluenceplatform.Unreachable:           rpcprotocol.CodePlatformUnreachable,
	}

	return rpcprotocol.Errorf(codes[failed.Failure], "%s", failed.Error())
}
