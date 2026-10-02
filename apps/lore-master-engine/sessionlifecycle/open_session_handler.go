package sessionlifecycle

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

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
		connected, err := confluenceplatform.Connect(ctx, confluenceplatform.ConnectOptions{
			BaseURL: params.BaseURL, Edition: params.Edition,
			SignIn: confluenceplatform.SignIn{
				Kind: params.Credential.Kind, Email: params.Credential.Email, Token: params.Credential.Token,
				User: params.Credential.User, Password: params.Credential.Password,
			},
			HTTPClient: environment.HTTPClient, Logger: environment.Logger,
		})
		if err != nil {
			return nil, connectError(err)
		}
		session := store.Add(Session{BaseURL: connected.BaseURL, Edition: connected.Edition, Platform: confluence{connected.Platform}})

		return rpcprotocol.SessionOpenResult{
			SessionID: session.ID, BaseURL: connected.BaseURL, Edition: connected.Edition, Version: connected.Version,
			User: rpcprotocol.SessionUser{DisplayName: connected.DisplayName, AccountID: connected.AccountID, Username: connected.Username},
		}, nil
	}
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
