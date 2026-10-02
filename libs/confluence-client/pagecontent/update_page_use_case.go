package pagecontent

import (
	"context"
	"errors"
	"net/http"

	"lore-master/libs/confluence-client/httptransport"
)

// UpdatePageInput replaces a page's title, parent and body in one call. ExpectedVersion
// is the version the change is based on; the update sends ExpectedVersion + 1. An empty
// ParentID leaves the parent unchanged; a different one moves the page.
type UpdatePageInput struct {
	ID              string
	ExpectedVersion int
	Title           string
	ParentID        string
	BodyStorage     string
	// Message is the version comment shown in the page history.
	Message string
}

// UpdatePage writes a new version of the page. A 409 (someone saved a newer version
// meanwhile) is a *VersionConflictError, a clashing title a *TitleTakenError, and a
// missing page a *PageNotFoundError.
func (p *Pages) UpdatePage(ctx context.Context, input UpdatePageInput) (Page, error) {
	if input.ID == "" || input.ExpectedVersion < 1 || input.Title == "" {
		return Page{}, errors.New("updating a page needs its id, the version it was read at, and a title")
	}
	page, err := p.api.update(ctx, input)
	var apiError *httptransport.APIError
	switch {
	case err == nil:
		return page, nil
	case errors.As(err, &apiError) && apiError.Status == http.StatusConflict:
		return Page{}, &VersionConflictError{ID: input.ID, ExpectedVersion: input.ExpectedVersion}
	}
	if taken := titleTaken(err, input.Title); taken != nil {
		return Page{}, taken
	}

	return Page{}, err
}
