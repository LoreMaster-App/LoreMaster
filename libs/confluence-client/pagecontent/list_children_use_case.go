package pagecontent

import "context"

// ListChildren lists a page's direct child pages, every page of them, in the order
// Confluence keeps them. A parent that does not exist is a *PageNotFoundError.
func (p *Pages) ListChildren(ctx context.Context, parentID string) ([]Page, error) {
	var children []Page
	err := p.api.children(ctx, parentID, func(child Page) error {
		children = append(children, child)

		return nil
	})

	return children, err
}
