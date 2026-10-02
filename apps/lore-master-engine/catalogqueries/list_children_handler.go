package catalogqueries

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/platformport"
)

// ListChildren handles page/children.
func ListChildren(sessions *sessionlifecycle.Store) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.PageChildrenParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if params.PageID == "" {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "pageId is required")
		}
		session, err := sessions.Get(params.SessionID)
		if err != nil {
			return nil, err
		}
		children, err := session.Platform.ListChildren(ctx, params.PageID)
		if err != nil {
			return nil, sessionlifecycle.PlatformError(err)
		}

		return pagesResult(children), nil
	}
}

func pagesResult(pages []platformport.RemotePage) rpcprotocol.PagesResult {
	result := rpcprotocol.PagesResult{Pages: make([]rpcprotocol.Page, len(pages))}
	for i, page := range pages {
		result.Pages[i] = rpcprotocol.Page{ID: page.ID, Title: page.Title, ParentID: page.ParentID, URL: page.URL}
	}

	return result
}
