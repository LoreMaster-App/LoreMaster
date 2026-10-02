package pagecontent

import (
	"context"
	"errors"
	"strings"
)

// errEnough stops a search once it has found what it was asked for.
var errEnough = errors.New("enough pages found")

// luceneSpecials are the characters CQL's text search treats as syntax. They are
// replaced by spaces so whatever a person types can never break the query.
var luceneSpecials = strings.NewReplacer(
	"+", " ", "-", " ", "&", " ", "|", " ", "!", " ", "(", " ", ")", " ", "{", " ", "}", " ",
	"[", " ", "]", " ", "^", " ", "~", " ", "*", " ", "?", " ", ":", " ", `\`, " ", "/", " ", `"`, " ",
)

// FindPages finds up to limit pages in the space whose title contains query, ignoring
// case, ordered by title: what a page picker shows as someone types. CQL's
// `title ~ "words*"` narrows the search on the server; its text match is looser
// than "contains" (stemming, word boundaries), so results are filtered here too. An
// empty query lists the space's pages. Whether every edition honours the trailing
// wildcard is to be confirmed against real sites (#29, #30).
func (p *Pages) FindPages(ctx context.Context, spaceKey string, query string, limit int) ([]Page, error) {
	cql := "space = " + cqlString(spaceKey) + " and type = page"
	words := strings.Join(strings.Fields(luceneSpecials.Replace(query)), " ")
	if words != "" {
		cql += " and title ~ " + cqlString(words+"*")
	}
	cql += " order by title"

	needle := strings.ToLower(strings.TrimSpace(query))
	var found []Page
	err := p.api.search(ctx, cql, func(page Page) error {
		if needle != "" && !strings.Contains(strings.ToLower(page.Title), needle) {
			return nil
		}
		found = append(found, page)
		if limit > 0 && len(found) >= limit {
			return errEnough
		}

		return nil
	})
	if errors.Is(err, errEnough) {
		err = nil
	}

	return found, err
}
