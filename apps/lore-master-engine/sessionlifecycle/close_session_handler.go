package sessionlifecycle

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// CloseSession handles session/close. An unknown id is not an error: the editor may
// close twice, or after the engine restarted.
func CloseSession(store *Store) rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SessionCloseParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		store.Remove(params.SessionID)

		return nil, nil
	}
}
