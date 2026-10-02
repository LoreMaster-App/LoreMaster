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
	create(ctx context.Context, input CreatePageInput) (Page, error)
	update(ctx context.Context, input UpdatePageInput) (Page, error)
	descendants(ctx context.Context, id string, visit func(Page) error) error
	trash(ctx context.Context, id string) error
	setProperty(ctx context.Context, id string, key string, value any) error
	addLabel(ctx context.Context, id string, label string) error
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

type storageBodyV2 struct {
	Representation string `json:"representation"`
	Value          string `json:"value"`
}

func (c cloudPages) create(ctx context.Context, input CreatePageInput) (Page, error) {
	request := map[string]any{
		"spaceId": input.SpaceID, "status": "current", "title": input.Title,
		"body": storageBodyV2{Representation: "storage", Value: input.BodyStorage},
	}
	if input.ParentID != "" {
		request["parentId"] = input.ParentID
	}
	var wire pageV2
	if err := c.client.PostJSON(ctx, "/api/v2/pages", request, &wire); err != nil {
		return Page{}, err
	}

	return wire.page(c.client.BaseURL()), nil
}

func (c cloudPages) update(ctx context.Context, input UpdatePageInput) (Page, error) {
	request := map[string]any{
		"id": input.ID, "status": "current", "title": input.Title,
		"body":    storageBodyV2{Representation: "storage", Value: input.BodyStorage},
		"version": map[string]any{"number": input.ExpectedVersion + 1, "message": input.Message},
	}
	if input.ParentID != "" {
		request["parentId"] = input.ParentID
	}
	var wire pageV2
	if err := c.client.PutJSON(ctx, "/api/v2/pages/"+url.PathEscape(input.ID), request, &wire); err != nil {
		return Page{}, notFound(err, input.ID)
	}

	return wire.page(c.client.BaseURL()), nil
}

func (c cloudPages) descendants(ctx context.Context, id string, visit func(Page) error) error {
	err := httptransport.PageV2(ctx, c.client, "/api/v2/pages/"+url.PathEscape(id)+"/descendants", url.Values{"limit": {"250"}}, func(wire pageV2) error {
		return visit(wire.page(c.client.BaseURL()))
	})

	return notFound(err, id)
}

func (c cloudPages) trash(ctx context.Context, id string) error {
	return notFound(c.client.Delete(ctx, "/api/v2/pages/"+url.PathEscape(id), nil), id)
}

type propertyV2 struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
}

// setProperty creates the content property, or updates it at its next version when it
// already exists (properties are versioned like pages).
func (c cloudPages) setProperty(ctx context.Context, id string, key string, value any) error {
	base := "/api/v2/pages/" + url.PathEscape(id) + "/properties"
	var existing struct {
		Results []propertyV2 `json:"results"`
	}
	if err := c.client.GetJSON(ctx, base, url.Values{"key": {key}}, &existing); err != nil {
		return notFound(err, id)
	}
	if len(existing.Results) == 0 {
		return c.client.PostJSON(ctx, base, map[string]any{"key": key, "value": value}, nil)
	}
	current := existing.Results[0]

	return c.client.PutJSON(ctx, base+"/"+url.PathEscape(current.ID), map[string]any{
		"key": key, "value": value, "version": map[string]any{"number": current.Version.Number + 1},
	}, nil)
}

func (c cloudPages) addLabel(ctx context.Context, id string, label string) error {
	return addLabelV1(ctx, c.client, id, label)
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

type storageBodyV1 struct {
	Storage struct {
		Value          string `json:"value"`
		Representation string `json:"representation"`
	} `json:"storage"`
}

func storageV1(value string) storageBodyV1 {
	var body storageBodyV1
	body.Storage.Value = value
	body.Storage.Representation = "storage"

	return body
}

// ancestorsV1 is how v1 says where a page goes: its parent as the only ancestor.
func ancestorsV1(parentID string) []map[string]string {
	if parentID == "" {
		return nil
	}

	return []map[string]string{{"id": parentID}}
}

func (s serverPages) create(ctx context.Context, input CreatePageInput) (Page, error) {
	request := map[string]any{
		"type": "page", "title": input.Title, "space": map[string]string{"key": input.SpaceKey},
		"body": storageV1(input.BodyStorage),
	}
	if ancestors := ancestorsV1(input.ParentID); ancestors != nil {
		request["ancestors"] = ancestors
	}
	var wire pageV1
	if err := s.client.PostJSON(ctx, "/rest/api/content", request, &wire); err != nil {
		return Page{}, err
	}
	page := wire.page(s.client.BaseURL())
	if page.ParentID == "" {
		page.ParentID = input.ParentID
	}

	return page, nil
}

func (s serverPages) update(ctx context.Context, input UpdatePageInput) (Page, error) {
	request := map[string]any{
		"id": input.ID, "type": "page", "title": input.Title,
		"body":    storageV1(input.BodyStorage),
		"version": map[string]any{"number": input.ExpectedVersion + 1, "message": input.Message},
	}
	if ancestors := ancestorsV1(input.ParentID); ancestors != nil {
		request["ancestors"] = ancestors
	}
	var wire pageV1
	if err := s.client.PutJSON(ctx, "/rest/api/content/"+url.PathEscape(input.ID), request, &wire); err != nil {
		return Page{}, notFound(err, input.ID)
	}
	page := wire.page(s.client.BaseURL())
	if page.ParentID == "" {
		page.ParentID = input.ParentID
	}

	return page, nil
}

func (s serverPages) descendants(ctx context.Context, id string, visit func(Page) error) error {
	err := httptransport.PageV1(ctx, s.client, "/rest/api/content/"+url.PathEscape(id)+"/descendant/page", url.Values{"expand": {"version,ancestors,space"}}, listPageSize, func(wire pageV1) error {
		return visit(wire.page(s.client.BaseURL()))
	})

	return notFound(err, id)
}

func (s serverPages) trash(ctx context.Context, id string) error {
	return notFound(s.client.Delete(ctx, "/rest/api/content/"+url.PathEscape(id), nil), id)
}

func (s serverPages) setProperty(ctx context.Context, id string, key string, value any) error {
	path := "/rest/api/content/" + url.PathEscape(id) + "/property/" + url.PathEscape(key)
	var existing struct {
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}
	err := s.client.GetJSON(ctx, path, nil, &existing)
	var apiError *httptransport.APIError
	if errors.As(err, &apiError) && apiError.Status == http.StatusNotFound {
		return notFound(s.client.PostJSON(ctx, "/rest/api/content/"+url.PathEscape(id)+"/property", map[string]any{"key": key, "value": value}, nil), id)
	}
	if err != nil {
		return err
	}

	return s.client.PutJSON(ctx, path, map[string]any{
		"key": key, "value": value, "version": map[string]any{"number": existing.Version.Number + 1},
	}, nil)
}

func (s serverPages) addLabel(ctx context.Context, id string, label string) error {
	return addLabelV1(ctx, s.client, id, label)
}

// addLabelV1 adds a global label through v1, which every edition serves (Cloud's v2
// can read labels but not add them). Adding a label the page already has is a no-op.
func addLabelV1(ctx context.Context, client *httptransport.Client, id string, label string) error {
	return notFound(client.PostJSON(ctx, "/rest/api/content/"+url.PathEscape(id)+"/label", []map[string]string{{"prefix": "global", "name": label}}, nil), id)
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
