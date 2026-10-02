package httptransport

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// maxPages bounds every listing, so a server that keeps answering with a next page
// fails the sync loudly instead of hanging it. 10 000 pages of 250 is 2.5 million items.
const maxPages = 10_000

// errTooManyPages is returned when a listing hits maxPages.
var errTooManyPages = fmt.Errorf("confluence kept returning more pages after %d; stopping", maxPages)

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
	for pages := 0; next != nil; pages++ {
		if pages == maxPages {
			return errTooManyPages
		}
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

// PageV1 lists every item of a REST v1 collection. It follows _links.next as given
// (relative to the base URL, or absolute on the same site), which covers both v1 paging
// styles: start/limit on Data Center and Server, and the cursor Cloud's v1 search
// uses. A page without a next link ends the walk if it is short, measured against the
// limit the server echoes (Data Center clamps it); a full page without one is followed
// by the next start, for servers that omit the link.
func PageV1[T any](ctx context.Context, c *Client, path string, query url.Values, limit int, visit func(T) error) error {
	firstQuery := url.Values{}
	for name, values := range query {
		firstQuery[name] = values
	}
	firstQuery.Set("start", "0")
	firstQuery.Set("limit", strconv.Itoa(limit))
	next := c.resolve(path, firstQuery)
	for start, pages := 0, 0; next != nil; pages++ {
		if pages == maxPages {
			return errTooManyPages
		}
		var page pageV1[T]
		if err := c.doJSON(ctx, "GET", next, nil, &page); err != nil {
			return err
		}
		for _, item := range page.Results {
			if err := visit(item); err != nil {
				return err
			}
		}
		size := max(page.Size, len(page.Results))
		start += size
		switch {
		case page.Links.Next != "":
			resolved, err := c.nextV1(page.Links.Next)
			if err != nil {
				return err
			}
			next = resolved
		case size == 0 || size < cmpPositive(page.Limit, limit):
			next = nil
		default:
			fallbackQuery := url.Values{}
			for name, values := range firstQuery {
				fallbackQuery[name] = values
			}
			fallbackQuery.Set("start", strconv.Itoa(start))
			next = c.resolve(path, fallbackQuery)
		}
	}

	return nil
}

// nextV1 resolves a v1 next link: a path relative to the base URL (context path
// included), or an absolute URL that must stay on the same site.
func (c *Client) nextV1(link string) (*url.URL, error) {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "//") {
		return c.sameOrigin(link)
	}

	return c.resolve(link, nil), nil
}

func cmpPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
}
