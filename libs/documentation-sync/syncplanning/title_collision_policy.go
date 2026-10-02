package syncplanning

import (
	"fmt"

	"lore-master/libs/documentation-sync/platformport"
)

// collisionDecision is what to do with an unannotated file whose title may already be
// taken on the platform.
type collisionDecision struct {
	kind  ActionKind
	page  platformport.RemotePage
	error string
}

// decideCollision applies titleCollision. No page with the title: create. One page and
// "adopt": take it over. Otherwise the sync stops for this file and says where the
// clashing page is, since creating would fail and overwriting a stranger's page
// silently would be worse.
func decideCollision(mode string, path string, title string, found []platformport.RemotePage) collisionDecision {
	switch {
	case len(found) == 0:
		return collisionDecision{kind: Create}
	case mode == "adopt" && len(found) == 1:
		return collisionDecision{kind: Adopt, page: found[0]}
	case mode == "adopt":
		return collisionDecision{error: fmt.Sprintf("%s: %d pages are titled %q; adopt needs exactly one", path, len(found), title)}
	}

	return collisionDecision{error: fmt.Sprintf(
		"%s: a page titled %q already exists (%s) and this file has no lore-master annotation; set titleCollision: adopt to take that page over, or give the file a \"title:\" override",
		path, title, found[0].URL)}
}
