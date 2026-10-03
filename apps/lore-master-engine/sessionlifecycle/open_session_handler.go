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

		connected, err := confluenceplatform.Connect(ctx, confluenceplatform.ConnectOptions{
			BaseURL: params.BaseURL, Edition: params.Edition,
			SignIn: confluenceplatform.SignIn{
				Kind: credential.Kind, Email: credential.Email, Token: credential.Token,
				User: credential.User, Password: credential.Password, AccessToken: credential.AccessToken,
			},
			HTTPClient: environment.HTTPClient, Logger: environment.Logger,
		})
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
