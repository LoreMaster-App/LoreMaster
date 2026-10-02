package catalogqueries

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/platformport"
)

// Search limits: a picker shows a screenful, and a person types more to narrow it.
const (
	defaultSearchLimit = 50
	maxSearchLimit     = 200
)

// SearchPages handles page/search.
func SearchPages(sessions *sessionlifecycle.Store) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.PageSearchParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if params.SpaceKey == "" {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "spaceKey is required")
		}
		limit := params.Limit
		if limit <= 0 {
			limit = defaultSearchLimit
		}
		limit = min(limit, maxSearchLimit)
		session, err := sessions.Get(params.SessionID)
		if err != nil {
			return nil, err
		}
		found, err := session.Platform.FindPages(ctx, platformport.SpaceRef{Key: params.SpaceKey}, params.Query, limit)
		if err != nil {
			return nil, sessionlifecycle.PlatformError(err)
		}

		return pagesResult(found), nil
	}
}
