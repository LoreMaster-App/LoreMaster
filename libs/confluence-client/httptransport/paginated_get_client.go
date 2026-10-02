package httptransport

import (
	"context"
	"net/url"
	"strconv"
)

// pageV2 is the envelope of a Cloud REST v2 list: results plus a cursor link.
type pageV2[T any] struct {
	Results []T `json:"results"`
	Links   struct {
		Next string `json:"next"`
	} `json:"_links"`
}

// PageV2 lists every item of a Cloud REST v2 collection, following _links.next (an
// absolute path on the same site) until there is none. visit returning an error stops
// the walk with that error.
func PageV2[T any](ctx context.Context, c *Client, path string, query url.Values, visit func(T) error) error {
	next := c.resolve(path, query)
	for next != nil {
		var page pageV2[T]
		if err := c.doJSON(ctx, "GET", next, nil, &page); err != nil {
			return err
		}
		for _, item := range page.Results {
			if err := visit(item); err != nil {
				return err
			}
		}
		next = nil
		if page.Links.Next != "" {
			resolved, err := c.sameOrigin(page.Links.Next)
			if err != nil {
				return err
			}
			next = resolved
		}
	}

	return nil
}

// pageV1 is the envelope of a REST v1 list.
type pageV1[T any] struct {
	Results []T `json:"results"`
	Size    int `json:"size"`
	Limit   int `json:"limit"`
	Links   struct {
		Next string `json:"next"`
	} `json:"_links"`
}

// PageV1 lists every item of a REST v1 collection with start/limit paging. It continues
// while the page carries a _links.next, which v1 sends whenever more results exist.
// Without one it falls back to "a short page is the last", measured against the limit
// the server echoes: Data Center clamps limit, so a page of 100 for a requested 250 is
// not the end.
func PageV1[T any](ctx context.Context, c *Client, path string, query url.Values, limit int, visit func(T) error) error {
	for start := 0; ; {
		pageQuery := url.Values{}
		for name, values := range query {
			pageQuery[name] = values
		}
		pageQuery.Set("start", strconv.Itoa(start))
		pageQuery.Set("limit", strconv.Itoa(limit))
		var page pageV1[T]
		if err := c.GetJSON(ctx, path, pageQuery, &page); err != nil {
			return err
		}
		for _, item := range page.Results {
			if err := visit(item); err != nil {
				return err
			}
		}
		size := max(page.Size, len(page.Results))
		if size == 0 {
			return nil
		}
		if page.Links.Next == "" {
			effectiveLimit := limit
			if page.Limit > 0 {
				effectiveLimit = page.Limit
			}
			if size < effectiveLimit {
				return nil
			}
		}
		start += size
	}
}
