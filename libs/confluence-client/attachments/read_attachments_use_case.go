package attachments

import (
	"context"
	"errors"
	"net/url"

	"lore-master/libs/confluence-client/httptransport"
)

// listPageSize is how many attachments a page of the v1 listing returns; a page rarely has
// more than a handful, so one request almost always suffices.
const listPageSize = 250

// ListAttachments lists a page's attachments for a two-way pull: each file's name, the content
// hash the sync stored in its comment (empty for a file the sync did not upload), and the
// site-relative path to download it.
func ListAttachments(ctx context.Context, client *httptransport.Client, pageID string) ([]Attachment, error) {
	if pageID == "" {
		return nil, errors.New("listing attachments needs the page id")
	}
	base := "/rest/api/content/" + url.PathEscape(pageID) + "/child/attachment"
	var out []Attachment
	err := httptransport.PageV1(ctx, client, base, url.Values{"expand": {"version"}}, listPageSize, func(a attachmentV1) error {
		out = append(out, Attachment{ID: a.ID, Filename: a.Title, Hash: a.comment(), DownloadPath: a.Links.Download})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// DownloadAttachment fetches an attachment's bytes from the download path ListAttachments gave.
// The path carries the version and modification date the server needs to serve the right bytes;
// the client splits that query off and resolves the path against the site.
func DownloadAttachment(ctx context.Context, client *httptransport.Client, downloadPath string) ([]byte, error) {
	if downloadPath == "" {
		return nil, errors.New("downloading an attachment needs its download path")
	}

	return client.GetBytes(ctx, downloadPath, nil, "")
}
