package pagecontent

import "context"

// ListDescendants lists every page below rootID, at any depth. With onlyMarked it lists
// only pages carrying MarkerLabel, through CQL
// `ancestor = "ID" and label = "lore-master" and type = page` on every edition (Cloud's
// v2 descendants carry no labels, and filtering on the server never fetches a
// hand-made page at all). Without it, Cloud uses v2 /pages/{id}/descendants and Data
// Center and Server v1 /content/{id}/descendant/page.
func (p *Pages) ListDescendants(ctx context.Context, rootID string, onlyMarked bool) ([]Page, error) {
	var found []Page
	collect := func(page Page) error {
		found = append(found, page)

		return nil
	}
	if onlyMarked {
		cql := "ancestor = " + cqlString(rootID) + " and label = " + cqlString(MarkerLabel) + " and type = page"

		return found, p.api.search(ctx, cql, collect)
	}

	return found, p.api.descendants(ctx, rootID, collect)
}
