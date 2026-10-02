package spacecatalog

import (
	"cmp"
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strings"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
)

// pageSize asks for large pages; the server may clamp it, which paging handles.
const pageSize = 250

// Options narrows the listing.
type Options struct {
	// IncludePersonal lists personal spaces too; by default only team spaces are listed.
	IncludePersonal bool
}

// ListSpaces lists every current space the user can see, sorted by name, then key.
// Cloud uses REST v2 (/api/v2/spaces, cursor paging); Data Center and Server use v1
// (/rest/api/space, start/limit paging, with the home page expanded). Personal spaces
// are filtered here rather than by the API's type parameter, which takes one value.
func ListSpaces(ctx context.Context, client *httptransport.Client, edition connection.Edition, options Options) ([]Space, error) {
	var spaces []Space
	keep := func(space Space) error {
		if !space.Personal || options.IncludePersonal {
			spaces = append(spaces, space)
		}

		return nil
	}

	var err error
	if edition == connection.Cloud {
		err = httptransport.PageV2(ctx, client, "/api/v2/spaces", url.Values{
			"status": {"current"}, "limit": {"250"},
		}, func(s spaceV2) error { return keep(s.space()) })
	} else {
		err = httptransport.PageV1(ctx, client, "/rest/api/space", url.Values{
			"status": {"current"}, "expand": {"homepage"},
		}, pageSize, func(s spaceV1) error { return keep(s.space()) })
	}
	if err != nil {
		return nil, err
	}
	slices.SortFunc(spaces, func(a, b Space) int {
		return cmp.Or(cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), cmp.Compare(a.Key, b.Key))
	})

	return spaces, nil
}

type spaceV2 struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	HomepageID string `json:"homepageId"`
}

func (s spaceV2) space() Space {
	return Space{ID: s.ID, Key: s.Key, Name: s.Name, HomepageID: s.HomepageID, Personal: s.Type == "personal"}
}

type spaceV1 struct {
	ID       json.Number `json:"id"`
	Key      string      `json:"key"`
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Homepage *struct {
		ID string `json:"id"`
	} `json:"homepage"`
}

func (s spaceV1) space() Space {
	space := Space{ID: s.ID.String(), Key: s.Key, Name: s.Name, Personal: s.Type == "personal"}
	if s.Homepage != nil {
		space.HomepageID = s.Homepage.ID
	}

	return space
}
