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
	prepared := Prepared{Pages: make(map[documentdiscovery.DocumentPath]PreparedPage, len(documents))}
	for _, document := range documents {
		converted := documentconversion.ConvertDocument(document, workspace)
		prepared.Warnings = append(prepared.Warnings, converted.Warnings...)
		page := PreparedPage{Converted: converted, Annotation: document.Annotation}
		for _, attachment := range converted.Attachments {
			content, err := read(attachment.Path)
			if err != nil {
				prepared.Warnings = append(prepared.Warnings, fmt.Sprintf("%s: the image %s cannot be read (%v); the page will show it broken", document.Path, attachment.Path, err))

				continue
			}
			page.Files = append(page.Files, PreparedFile{Name: attachment.Filename, Path: attachment.Path, Hash: attachmentHash(content)})
		}
		prepared.Pages[document.Path] = page
	}

	return prepared
}
