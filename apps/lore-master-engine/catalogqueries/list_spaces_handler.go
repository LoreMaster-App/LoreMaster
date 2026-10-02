package catalogqueries

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
)

// ListSpaces handles space/list.
func ListSpaces(sessions *sessionlifecycle.Store) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SpaceListParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		session, err := sessions.Get(params.SessionID)
		if err != nil {
			return nil, err
		}
		spaces, err := session.Platform.ListSpaces(ctx)
		if err != nil {
			return nil, sessionlifecycle.PlatformError(err)
		}
		result := rpcprotocol.SpaceListResult{Spaces: make([]rpcprotocol.Space, len(spaces))}
		for i, space := range spaces {
			result.Spaces[i] = rpcprotocol.Space{ID: space.ID, Key: space.Key, Name: space.Name, HomepageID: space.HomepageID}
		}

		return result, nil
	}
}
