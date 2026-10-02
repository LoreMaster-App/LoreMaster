package sessionlifecycle

import (
	"context"
	"errors"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/libs/documentation-sync/platformport"
)

// PlatformError gives a platform failure the code the editor acts on: a missing page,
// a taken title and a version conflict each have their own; a cancellation stays one;
// anything else is the platform not answering as expected, with its message.
func PlatformError(err error) error {
	var missing *platformport.PageNotFoundError
	var taken *platformport.TitleTakenError
	var conflict *platformport.VersionConflictError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return err
	case errors.As(err, &missing):
		return rpcprotocol.Errorf(rpcprotocol.CodeNotFound, "%s", err.Error())
	case errors.As(err, &taken):
		return rpcprotocol.Errorf(rpcprotocol.CodeTitleTaken, "%s", err.Error())
	case errors.As(err, &conflict):
		return rpcprotocol.Errorf(rpcprotocol.CodeVersionConflict, "%s", err.Error())
	}

	return rpcprotocol.Errorf(rpcprotocol.CodePlatformUnreachable, "%s", err.Error())
}
