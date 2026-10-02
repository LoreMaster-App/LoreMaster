package attachmentupload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime"
	"net/url"
	"path"

	"lore-master/libs/confluence-client/httptransport"
)

// attachmentV1 is an attachment as v1 returns it. The comment sits in metadata on some
// editions and in extensions on others, so both are read.
type attachmentV1 struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Metadata struct {
		Comment string `json:"comment"`
	} `json:"metadata"`
	Extensions struct {
		Comment string `json:"comment"`
	} `json:"extensions"`
}

func (a attachmentV1) comment() string {
	if a.Metadata.Comment != "" {
		return a.Metadata.Comment
	}

	return a.Extensions.Comment
}

// ContentHash is the "sha256:<hex>" fingerprint stored in an attachment's comment.
func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)

	return "sha256:" + hex.EncodeToString(sum[:])
}

// UploadAttachment makes sure the page has input as an attachment, uploading only when
// needed:
//
//   - no attachment of that file name yet: POST /child/attachment creates it;
//   - one with the same content hash in its comment: nothing is sent (Skipped);
//   - one with different content: POST /child/attachment/{id}/data adds a new version,
//     which keeps the attachment id, so pages that show it keep working.
//
// Every upload is a minor edit, so watchers are not notified per image.
func UploadAttachment(ctx context.Context, client *httptransport.Client, pageID string, input AttachmentInput) (Attachment, error) {
	if pageID == "" || input.Filename == "" {
		return Attachment{}, errors.New("uploading an attachment needs the page id and a file name")
	}
	hash := ContentHash(input.Content)
	base := "/rest/api/content/" + url.PathEscape(pageID) + "/child/attachment"

	var existing struct {
		Results []attachmentV1 `json:"results"`
	}
	if err := client.GetJSON(ctx, base, url.Values{"filename": {input.Filename}, "expand": {"version"}}, &existing); err != nil {
		return Attachment{}, err
	}
	var current *attachmentV1
	for i := range existing.Results {
		if existing.Results[i].Title == input.Filename {
			current = &existing.Results[i]

			break
		}
	}
	if current != nil && current.comment() == hash {
		return Attachment{ID: current.ID, Filename: input.Filename, Hash: hash, Skipped: true}, nil
	}

	contentType := input.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(input.Filename))
	}
	files := []httptransport.MultipartFile{{Field: "file", FileName: input.Filename, ContentType: contentType, Content: input.Content}}
	fields := map[string]string{"comment": hash, "minorEdit": "true"}

	target := base
	if current != nil {
		target = base + "/" + url.PathEscape(current.ID) + "/data"
	}
	var raw json.RawMessage
	if err := client.PostMultipart(ctx, target, fields, files, &raw); err != nil {
		return Attachment{}, err
	}
	uploaded, err := decodeUploaded(raw)
	if err != nil {
		return Attachment{}, err
	}

	return Attachment{ID: uploaded.ID, Filename: input.Filename, Hash: hash}, nil
}

// decodeUploaded reads either answer shape: creating returns {"results": [attachment]},
// updating the data returns the attachment itself.
func decodeUploaded(raw json.RawMessage) (attachmentV1, error) {
	var list struct {
		Results []attachmentV1 `json:"results"`
	}
	if err := json.Unmarshal(raw, &list); err == nil && len(list.Results) > 0 {
		return list.Results[0], nil
	}
	var single attachmentV1
	if err := json.Unmarshal(raw, &single); err != nil || single.ID == "" {
		return attachmentV1{}, errors.New("confluence accepted the upload but did not return the attachment")
	}

	return single, nil
}
