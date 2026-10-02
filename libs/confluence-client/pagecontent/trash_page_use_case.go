package pagecontent

import (
	"context"
	"errors"
)

// TrashPage moves a page to the space's trash (v2 and v1 DELETE both trash a current
// page rather than purge it), from where a space admin can restore it. A page that is
// already gone counts as trashed, so prune can be re-run safely.
func (p *Pages) TrashPage(ctx context.Context, id string) error {
	err := p.api.trash(ctx, id)
	var notFound *PageNotFoundError
	if errors.As(err, &notFound) {
		return nil
	}

	return err
}
