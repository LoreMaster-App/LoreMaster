package pagecontent

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"lore-master/libs/confluence-client/httptransport"
)

// CreatePageInput is a new page. SpaceKey is required on every edition (v1 creates by
// key, and recovery searches by it); SpaceID is required on Cloud, whose v2 API creates
// by id. An empty ParentID creates a top-level page.
type CreatePageInput struct {
	SpaceID     string
	SpaceKey    string
	ParentID    string
	Title       string
	BodyStorage string
}

// CreatePage creates a page and returns it. A title another page already has is a
// *TitleTakenError. When the request ends in 502, 503 or 504 the page may have been
// created anyway (the transport never repeats a POST), so CreatePage looks for the
// title in the space: one exact match under the same parent is returned as the
// created page; otherwise the original error stands.
func (p *Pages) CreatePage(ctx context.Context, input CreatePageInput) (Page, error) {
	if input.SpaceKey == "" || input.Title == "" {
		return Page{}, errors.New("creating a page needs a space key and a title")
	}
	page, err := p.api.create(ctx, input)
	if err == nil {
		return page, nil
	}
	if taken := titleTaken(err, input.Title); taken != nil {
		return Page{}, taken
	}
	var apiError *httptransport.APIError
	if errors.As(err, &apiError) && isGatewayFailure(apiError.Status) {
		if recovered, ok := p.recoverCreated(ctx, input); ok {
			return recovered, nil
		}
	}

	return Page{}, err
}

func (p *Pages) recoverCreated(ctx context.Context, input CreatePageInput) (Page, bool) {
	found, err := p.SearchPages(ctx, input.SpaceKey, input.Title)
	if err != nil || len(found) != 1 || found[0].Title != input.Title {
		return Page{}, false
	}
	if input.ParentID != "" && found[0].ParentID != input.ParentID {
		return Page{}, false
	}

	return found[0], true
}

func isGatewayFailure(status int) bool {
	return status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

// titleTaken recognises Confluence's "A page with this title already exists" answer,
// a 400 on every edition.
func titleTaken(err error, title string) *TitleTakenError {
	var apiError *httptransport.APIError
	if !errors.As(err, &apiError) || apiError.Status != http.StatusBadRequest {
		return nil
	}
	message := strings.ToLower(apiError.Body)
	if strings.Contains(message, "title") && strings.Contains(message, "already exist") {
		return &TitleTakenError{Title: title, Message: apiError.Body}
	}

	return nil
}
