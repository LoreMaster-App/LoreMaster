package pagecontent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
)

// listPageSize asks for large pages; the server may clamp it.
const listPageSize = 250

// pageAPI is everything the use cases need from Confluence, in one of two dialects.
type pageAPI interface {
	get(ctx context.Context, id string, withBody bool) (Page, error)
	children(ctx context.Context, id string, visit func(Page) error) error
	search(ctx context.Context, cql string, visit func(Page) error) error
}

// Pages reads and writes pages on one site.
type Pages struct {
	api pageAPI
}

// New returns the Pages for a site: REST v2 on Cloud, v1 on Data Center and Server.
func New(client *httptransport.Client, edition connection.Edition) *Pages {
	if edition == connection.Cloud {
		return &Pages{api: cloudPages{client: client}}
	}

	return &Pages{api: serverPages{client: client}}
}

// cloudPages speaks Cloud REST v2 for pages, and v1 for CQL search, which v2 lacks.
type cloudPages struct {
	client *httptransport.Client
}

type pageV2 struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SpaceID  string `json:"spaceId"`
	ParentID string `json:"parentId"`
	Version  struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Links struct {
		WebUI string `json:"webui"`
		Base  string `json:"base"`
	} `json:"_links"`
}

func (p pageV2) page(siteBase string) Page {
	return Page{
		ID: p.ID, Title: p.Title, SpaceID: p.SpaceID, ParentID: p.ParentID,
		Version: p.Version.Number, BodyStorage: p.Body.Storage.Value,
		WebURL: webURL(p.Links.Base, siteBase, p.Links.WebUI),
	}
}

func (c cloudPages) get(ctx context.Context, id string, withBody bool) (Page, error) {
	query := url.Values{}
	if withBody {
		query.Set("body-format", "storage")
	}
	var wire pageV2
	if err := c.client.GetJSON(ctx, "/api/v2/pages/"+url.PathEscape(id), query, &wire); err != nil {
		return Page{}, notFound(err, id)
	}

	return wire.page(c.client.BaseURL()), nil
}

func (c cloudPages) children(ctx context.Context, id string, visit func(Page) error) error {
	err := httptransport.PageV2(ctx, c.client, "/api/v2/pages/"+url.PathEscape(id)+"/children", url.Values{"limit": {"250"}}, func(wire pageV2) error {
		child := wire.page(c.client.BaseURL())
		child.ParentID = id

		return visit(child)
	})

	return notFound(err, id)
}

func (c cloudPages) search(ctx context.Context, cql string, visit func(Page) error) error {
	return searchV1(ctx, c.client, cql, visit)
}

// serverPages speaks REST v1, the only API of Data Center and Server.
type serverPages struct {
	client *httptransport.Client
}

type pageV1 struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Space struct {
		ID  json.Number `json:"id"`
		Key string      `json:"key"`
	} `json:"space"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Ancestors []struct {
		ID string `json:"id"`
	} `json:"ancestors"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Links struct {
		WebUI string `json:"webui"`
		Base  string `json:"base"`
	} `json:"_links"`
}

func (p pageV1) page(siteBase string) Page {
	page := Page{
		ID: p.ID, Title: p.Title, SpaceID: p.Space.ID.String(), SpaceKey: p.Space.Key,
		Version: p.Version.Number, BodyStorage: p.Body.Storage.Value,
		WebURL: webURL(p.Links.Base, siteBase, p.Links.WebUI),
	}
	// v1 lists ancestors root first; the parent is the last one.
	if len(p.Ancestors) > 0 {
		page.ParentID = p.Ancestors[len(p.Ancestors)-1].ID
	}

	return page
}

func (s serverPages) get(ctx context.Context, id string, withBody bool) (Page, error) {
	expand := "version,ancestors,space"
	if withBody {
		expand += ",body.storage"
	}
	var wire pageV1
	if err := s.client.GetJSON(ctx, "/rest/api/content/"+url.PathEscape(id), url.Values{"expand": {expand}}, &wire); err != nil {
		return Page{}, notFound(err, id)
	}

	return wire.page(s.client.BaseURL()), nil
}

func (s serverPages) children(ctx context.Context, id string, visit func(Page) error) error {
	err := httptransport.PageV1(ctx, s.client, "/rest/api/content/"+url.PathEscape(id)+"/child/page", url.Values{"expand": {"version,space"}}, listPageSize, func(wire pageV1) error {
		child := wire.page(s.client.BaseURL())
		child.ParentID = id

		return visit(child)
	})

	return notFound(err, id)
}

func (s serverPages) search(ctx context.Context, cql string, visit func(Page) error) error {
	return searchV1(ctx, s.client, cql, visit)
}

// searchV1 runs CQL through /rest/api/content/search, which every edition serves.
func searchV1(ctx context.Context, client *httptransport.Client, cql string, visit func(Page) error) error {
	return httptransport.PageV1(ctx, client, "/rest/api/content/search", url.Values{
		"cql": {cql}, "expand": {"version,ancestors,space"},
	}, listPageSize, func(wire pageV1) error { return visit(wire.page(client.BaseURL())) })
}

// webURL makes a page's webui link absolute against the response's base, or the
// site's when the response has none.
func webURL(responseBase string, siteBase string, webUI string) string {
	if webUI == "" {
		return ""
	}
	base := responseBase
	if base == "" {
		base = siteBase
	}

	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(webUI, "/")
}

func notFound(err error, id string) error {
	var apiError *httptransport.APIError
	if errors.As(err, &apiError) && apiError.Status == http.StatusNotFound {
		return &PageNotFoundError{ID: id}
	}

	return err
}
