package documentconversion

import (
	"fmt"
	"slices"

	"github.com/yuin/goldmark/ast"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// ConvertDocument maps a parsed document onto a platform-neutral page body. It is pure:
// it reads no file and asks no platform, so the same document always converts the
// same way. What cannot be carried over (a link to a file outside the sync, an HTML
// block) is reported in Warnings, never dropped silently.
func ConvertDocument(document documentparsing.MarkdownDocument, workspace Workspace) Converted {
	inventory := documentparsing.InventoryLinks(document)
	c := &converter{
		document:  document,
		source:    document.Body,
		workspace: workspace,
		pageLinks: map[ast.Node]documentparsing.PageLink{},
		images:    map[ast.Node][]documentparsing.ImageRef{},
		names:     map[documentdiscovery.DocumentPath]string{},
		used:      map[documentdiscovery.DocumentPath]bool{},
	}
	for _, link := range inventory.PageLinks {
		c.pageLinks[link.Node] = link
	}
	var local []documentdiscovery.DocumentPath
	for _, image := range inventory.Images {
		c.images[image.Node] = append(c.images[image.Node], image)
		if image.Path != "" {
			local = append(local, image.Path)
		}
	}
	// Names are given over every image so they do not shift when one is commented out;
	// only the images that reach the page are attached.
	named := nameAttachments(local)
	for _, attachment := range named {
		c.names[attachment.Path] = attachment.Filename
	}
	for _, warning := range inventory.Warnings {
		c.warn("%s", warning)
	}

	converted := Converted{Document: platformport.Document{Blocks: c.blocks(document.AST)}, Warnings: c.warnings}
	converted.RenderHash = renderHash(converted.Document)
	converted.LinkedPages = c.pages
	for _, attachment := range named {
		if c.used[attachment.Path] {
			converted.Attachments = append(converted.Attachments, attachment)
		}
	}

	return converted
}

// converter carries one document's lookups while its tree is mapped.
type converter struct {
	document  documentparsing.MarkdownDocument
	source    []byte
	workspace Workspace
	pageLinks map[ast.Node]documentparsing.PageLink
	images    map[ast.Node][]documentparsing.ImageRef
	names     map[documentdiscovery.DocumentPath]string
	used      map[documentdiscovery.DocumentPath]bool
	pages     []documentdiscovery.DocumentPath
	warnings  []string
}

// linked records a page this one links to.
func (c *converter) linked(path documentdiscovery.DocumentPath) {
	if !slices.Contains(c.pages, path) {
		c.pages = append(c.pages, path)
	}
}

func (c *converter) warn(format string, args ...any) {
	c.warnings = append(c.warnings, fmt.Sprintf("%s: ", c.document.Path)+fmt.Sprintf(format, args...))
}

// imageSource is where an inventoried image comes from: an attachment of this page or
// a URL.
func (c *converter) imageSource(image documentparsing.ImageRef) platformport.ImageSource {
	if image.URL != "" {
		return &platformport.URLRef{URL: image.URL}
	}
	c.used[image.Path] = true

	return &platformport.AttachmentRef{Filename: c.names[image.Path]}
}
