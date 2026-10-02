package rpcprotocol

// MethodSpaceList lists the spaces the session's user can see, sorted by name.
const MethodSpaceList = "space/list"

// SpaceListParams asks for the spaces.
type SpaceListParams struct {
	SessionID string `json:"sessionId"`
	// IncludePersonal adds personal spaces to the team spaces.
	IncludePersonal bool `json:"includePersonal,omitempty"`
}

// SpaceListResult is every space, already complete: the engine follows the
// platform's paging itself.
type SpaceListResult struct {
	Spaces []Space `json:"spaces"`
}

// Space is one space.
type Space struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	// HomepageID is the space's home page, the default parent page; empty when none.
	HomepageID string `json:"homepageId,omitempty"`
	Personal   bool   `json:"personal,omitempty"`
}
