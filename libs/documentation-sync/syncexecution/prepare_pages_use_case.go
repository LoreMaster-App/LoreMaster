package syncexecution

import (
	"fmt"

	"lore-master/libs/documentation-sync/documentconversion"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// PreparePages converts every document and hashes the files its page shows. titles
// are the final page titles (documenttree.PageTitles).
func PreparePages(documents []documentparsing.MarkdownDocument, titles map[documentdiscovery.DocumentPath]string, read FileReader) Prepared {
	workspace := documentconversion.NewWorkspace(documents, titles)
	prepared := Prepared{Pages: make(map[documentdiscovery.DocumentPath]PreparedPage, len(documents)), Workspace: workspace}
	for _, document := range documents {
		page, warnings := preparePage(document, workspace, read)
		prepared.Warnings = append(prepared.Warnings, warnings...)
		prepared.Pages[document.Path] = page
	}

	return prepared
}

// preparePage converts one document against the workspace and hashes the attachments its page
// shows. A two-way pull reuses it so a pulled body gets the very render and attachment hashes
// the next sync will compute, which is what makes a pulled file read back as unchanged.
func preparePage(document documentparsing.MarkdownDocument, workspace documentconversion.Workspace, read FileReader) (PreparedPage, []string) {
	converted := documentconversion.ConvertDocument(document, workspace)
	warnings := append([]string(nil), converted.Warnings...)
	page := PreparedPage{Converted: converted, Annotation: document.Annotation}
	for _, attachment := range converted.Attachments {
		content, err := read(attachment.Path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: the image %s cannot be read (%v); the page will show it broken", document.Path, attachment.Path, err))

			continue
		}
		page.Files = append(page.Files, PreparedFile{Name: attachment.Filename, Path: attachment.Path, Hash: attachmentHash(content)})
	}

	return page, warnings
}
