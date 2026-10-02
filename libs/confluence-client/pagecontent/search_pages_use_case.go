package pagecontent

import (
	"context"
	"strings"
)

// SearchPages finds the pages titled title in the space, through CQL
// `space = "KEY" and title = "T" and type = page`, with both values escaped. CQL's title
// match is looser than equality, so results are filtered here to titles equal to title
// ignoring case, the same comparison the duplicate-title validator uses: whether
// Confluence itself treats titles differing only by case as a clash is unverified
// (#29/#30), and the stricter reading is the safe one.
func (p *Pages) SearchPages(ctx context.Context, spaceKey string, title string) ([]Page, error) {
	cql := "space = " + cqlString(spaceKey) + " and title = " + cqlString(title) + " and type = page"
	var matches []Page
	err := p.api.search(ctx, cql, func(page Page) error {
		if strings.EqualFold(page.Title, title) {
			matches = append(matches, page)
		}

		return nil
	})

	return matches, err
}
