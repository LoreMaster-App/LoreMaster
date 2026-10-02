package attachmentupload

// Attachment is one file on a page.
type Attachment struct {
	ID       string
	Filename string
	// Hash is "sha256:<hex>" of the content, kept in the attachment's comment so a
	// re-sync can tell an unchanged file without downloading it.
	Hash string
	// Skipped is true when the page already had this file with this content and
	// nothing was uploaded.
	Skipped bool
}

// AttachmentInput is a file to upload.
type AttachmentInput struct {
	Filename string
	// ContentType defaults to the type of the file name's extension.
	ContentType string
	Content     []byte
}
