package editiondetect

import (
	"context"
	"errors"
	"log/slog"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/confluenceplatform"
)

// DetectEdition handles edition/detect: probe the site and report its edition, with no
// credential and no session.
func DetectEdition(logger *slog.Logger) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.EditionDetectParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		detected, err := confluenceplatform.Detect(ctx, confluenceplatform.DetectOptions{BaseURL: params.BaseURL, Logger: logger})
		if err != nil {
			return nil, detectError(err)
		}

		return rpcprotocol.EditionDetectResult{BaseURL: detected.BaseURL, Edition: detected.Edition, Version: detected.Version}, nil
	}
}

// detectError turns a Detect failure into the editor's error code. Only a bad address or
// an unreachable/unrecognised site can fail a credential-less probe.
func detectError(err error) error {
	var failed *confluenceplatform.ConnectError
	if !errors.As(err, &failed) {
		return err
	}
	var code int64 = rpcprotocol.CodePlatformUnreachable
	if failed.Failure == confluenceplatform.InvalidAddress {
		code = rpcprotocol.CodeInvalidParams
	}

	return rpcprotocol.Errorf(code, "%s", failed.Error())
}
