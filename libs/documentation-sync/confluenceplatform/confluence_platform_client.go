package confluenceplatform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"lore-master/libs/confluence-client/attachments"
	"lore-master/libs/confluence-client/authentication"
	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
	"lore-master/libs/confluence-client/pagecontent"
	"lore-master/libs/confluence-client/spacecatalog"
	"lore-master/libs/confluence-client/storageformat"
	"lore-master/libs/documentation-sync/platformport"
)

// Options configures the adapter.
type Options struct {
	Connection connection.Connection
	Credential authentication.Credential
	// Render sets the Mermaid and link modes (the settings file's mermaidMode and
	// linkMode).
	Render storageformat.Options
	// HTTPClient and Logger default as in httptransport.
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// Platform is a Confluence site behind the sync's port.
type Platform struct {
	client  *httptransport.Client
	pages   *pagecontent.Pages
	edition connection.Edition
	render  storageformat.Options
}

var _ platformport.DocumentationPlatform = (*Platform)(nil)

// New connects the adapter. A credential the edition cannot accept is refused here,
// with its remedy, before any request.
func New(options Options) (*Platform, error) {
	if err := authentication.Supports(options.Connection.Edition, options.Connection.Version, options.Credential); err != nil {
		return nil, err
	}
	header, err := authentication.AuthorizationHeader(options.Credential)
	if err != nil {
		return nil, err
	}
	client, err := httptransport.New(httptransport.Options{
		BaseURL: options.Connection.BaseURL, AuthorizationHeader: header,
		HTTPClient: options.HTTPClient, Logger: options.Logger,
	})
	if err != nil {
		return nil, err
	}

	return &Platform{
		client: client, pages: pagecontent.New(client, options.Connection.Edition),
		edition: options.Connection.Edition, render: options.Render,
	}, nil
}

// ListSpaces implements platformport.DocumentationPlatform.
func (p *Platform) ListSpaces(ctx context.Context) ([]platformport.Space, error) {
	spaces, err := spacecatalog.ListSpaces(ctx, p.client, p.edition, spacecatalog.Options{})
	if err != nil {
		return nil, err
	}
	out := make([]platformport.Space, len(spaces))
	for i, space := range spaces {
		out[i] = platformport.Space{ID: space.ID, Key: space.Key, Name: space.Name, HomepageID: space.HomepageID}
	}

	return out, nil
}

// GetPage implements platformport.DocumentationPlatform.
func (p *Platform) GetPage(ctx context.Context, id string) (platformport.RemotePage, error) {
	page, err := p.pages.GetPage(ctx, id, false)

	return remote(page), portError(err)
}

// GetPageContent implements platformport.DocumentationPlatform: it fetches the page with its
// storage-format body and reverses the render — parse the storage into a Document, map it onto
// the neutral model — for a two-way pull. Flags from the parser name anything the storage
// format carried that could not be converted faithfully.
func (p *Platform) GetPageContent(ctx context.Context, id string) (platformport.PageContent, error) {
	page, err := p.pages.GetPage(ctx, id, true)
	if err != nil {
		return platformport.PageContent{}, portError(err)
	}
	document, flags, err := storageformat.Parse(page.BodyStorage)
	if err != nil {
		return platformport.PageContent{}, err
	}

	return platformport.PageContent{Version: page.Version, Body: fromStorage(document), Flags: flags}, nil
}

// ListChildren implements platformport.DocumentationPlatform.
func (p *Platform) ListChildren(ctx context.Context, parentID string) ([]platformport.RemotePage, error) {
	children, err := p.pages.ListChildren(ctx, parentID)

	return remotes(children), portError(err)
}

// FindPagesByTitle implements platformport.DocumentationPlatform.
func (p *Platform) FindPagesByTitle(ctx context.Context, space platformport.SpaceRef, title string) ([]platformport.RemotePage, error) {
	found, err := p.pages.SearchPages(ctx, space.Key, title)

	return remotes(found), portError(err)
}

// FindPages implements platformport.DocumentationPlatform.
func (p *Platform) FindPages(ctx context.Context, space platformport.SpaceRef, query string, limit int) ([]platformport.RemotePage, error) {
	found, err := p.pages.FindPages(ctx, space.Key, query, limit)

	return remotes(found), portError(err)
}

// CreatePage implements platformport.DocumentationPlatform.
func (p *Platform) CreatePage(ctx context.Context, page platformport.NewPage) (platformport.RemotePage, error) {
	body, err := p.renderBody(page.Body)
	if err != nil {
		return platformport.RemotePage{}, err
	}
	created, err := p.pages.CreatePage(ctx, pagecontent.CreatePageInput{
		SpaceID: page.Space.ID, SpaceKey: page.Space.Key, ParentID: page.ParentID, Title: page.Title, BodyStorage: body,
	})

	return remote(created), portError(err)
}

// UpdatePage implements platformport.DocumentationPlatform.
func (p *Platform) UpdatePage(ctx context.Context, update platformport.PageUpdate) (platformport.RemotePage, error) {
	body, err := p.renderBody(update.Body)
	if err != nil {
		return platformport.RemotePage{}, err
	}
	updated, err := p.pages.UpdatePage(ctx, pagecontent.UpdatePageInput{
		ID: update.ID, ExpectedVersion: update.ExpectedVersion, Title: update.Title, ParentID: update.ParentID,
		BodyStorage: body, Message: update.Message,
	})

	return remote(updated), portError(err)
}

// MarkPage implements platformport.DocumentationPlatform.
func (p *Platform) MarkPage(ctx context.Context, id string, sourcePath string) error {
	return portError(p.pages.MarkPage(ctx, id, sourcePath))
}

// ListMarkedDescendants implements platformport.DocumentationPlatform.
func (p *Platform) ListMarkedDescendants(ctx context.Context, rootID string) ([]platformport.RemotePage, error) {
	found, err := p.pages.ListDescendants(ctx, rootID, true)

	return remotes(found), portError(err)
}

// TrashPage implements platformport.DocumentationPlatform.
func (p *Platform) TrashPage(ctx context.Context, id string) error {
	return portError(p.pages.TrashPage(ctx, id))
}

// UploadFile implements platformport.DocumentationPlatform.
func (p *Platform) UploadFile(ctx context.Context, pageID string, file platformport.File) (platformport.UploadedFile, error) {
	uploaded, err := attachments.UploadAttachment(ctx, p.client, pageID, attachments.AttachmentInput{
		Filename: file.Name, ContentType: file.ContentType, Content: file.Content,
	})
	if err != nil {
		return platformport.UploadedFile{}, portError(err)
	}

	return platformport.UploadedFile{ID: uploaded.ID, Name: uploaded.Filename, Hash: uploaded.Hash, Skipped: uploaded.Skipped}, nil
}

// ListAttachments implements platformport.DocumentationPlatform.
func (p *Platform) ListAttachments(ctx context.Context, pageID string) ([]platformport.RemoteAttachment, error) {
	list, err := attachments.ListAttachments(ctx, p.client, pageID)
	if err != nil {
		return nil, portError(err)
	}
	out := make([]platformport.RemoteAttachment, 0, len(list))
	for _, file := range list {
		out = append(out, platformport.RemoteAttachment{Filename: file.Filename, Hash: file.Hash})
	}

	return out, nil
}

// DownloadAttachment implements platformport.DocumentationPlatform.
func (p *Platform) DownloadAttachment(ctx context.Context, pageID string, filename string) ([]byte, error) {
	list, err := attachments.ListAttachments(ctx, p.client, pageID)
	if err != nil {
		return nil, portError(err)
	}
	for _, file := range list {
		if file.Filename == filename {
			content, err := attachments.DownloadAttachment(ctx, p.client, file.DownloadPath)

			return content, portError(err)
		}
	}

	return nil, fmt.Errorf("the page %s has no attachment named %q", pageID, filename)
}

func (p *Platform) renderBody(doc platformport.Document) (string, error) {
	mapped, err := toStorage(doc)
	if err != nil {
		return "", err
	}

	return storageformat.Render(mapped, p.render)
}

func remote(page pagecontent.Page) platformport.RemotePage {
	return platformport.RemotePage{ID: page.ID, Title: page.Title, ParentID: page.ParentID, Version: page.Version, URL: page.WebURL}
}

func remotes(pages []pagecontent.Page) []platformport.RemotePage {
	out := make([]platformport.RemotePage, len(pages))
	for i, page := range pages {
		out[i] = remote(page)
	}

	return out
}

// portError translates the client's typed errors into the port's, so the sync never
// needs to know which platform answered.
func portError(err error) error {
	var notFound *pagecontent.PageNotFoundError
	var conflict *pagecontent.VersionConflictError
	var taken *pagecontent.TitleTakenError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &notFound):
		return &platformport.PageNotFoundError{ID: notFound.ID}
	case errors.As(err, &conflict):
		return &platformport.VersionConflictError{ID: conflict.ID, ExpectedVersion: conflict.ExpectedVersion}
	case errors.As(err, &taken):
		return &platformport.TitleTakenError{Title: taken.Title}
	}

	return err
}
