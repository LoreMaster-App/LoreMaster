package pagecontent

import "context"

// GetPage reads one page; withBody also fetches its storage-format body. A page that
// does not exist, or that the user cannot see, is a *PageNotFoundError.
func (p *Pages) GetPage(ctx context.Context, id string, withBody bool) (Page, error) {
	return p.api.get(ctx, id, withBody)
}
