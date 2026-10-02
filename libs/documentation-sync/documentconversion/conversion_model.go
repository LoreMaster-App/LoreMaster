package documentconversion

import (
	"github.com/yuin/goldmark/ast"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// Workspace is what a document's links can reach: every document being synced, with
// its final page title and its headings.
type Workspace struct {
	lookup  documentdiscovery.Discovery
	targets map[documentdiscovery.DocumentPath]target
}

type target struct {
	title        string
	titleHeading *ast.Heading
	headings     []documentparsing.Heading
}

// NewWorkspace indexes the documents being synced. titles are their final page titles
// (prefix included), as documenttree.PageTitles returns them.
func NewWorkspace(documents []documentparsing.MarkdownDocument, titles map[documentdiscovery.DocumentPath]string) Workspace {
	paths := make([]documentdiscovery.DocumentPath, 0, len(documents))
	targets := make(map[documentdiscovery.DocumentPath]target, len(documents))
	for _, document := range documents {
		paths = append(paths, document.Path)
		targets[document.Path] = target{title: titles[document.Path], titleHeading: document.TitleHeading, headings: documentparsing.Headings(document)}
	}

	return Workspace{lookup: documentdiscovery.NewDiscovery(paths, nil), targets: targets}
}

// Converted is one document ready for a platform.
type Converted struct {
	Document platformport.Document
	// RenderHash identifies Document: it changes whenever what the page shows changes,
	// including through another file (a link that now resolves, a renamed heading).
	RenderHash string
	// LinkedPages are the synced documents this page links to, in order, without
	// repeats; the page itself is one when it links to its own headings.
	LinkedPages []documentdiscovery.DocumentPath
	// Attachments are the local files the page needs, each under the name the page
	// refers to it by, in document order and without repeats.
	Attachments []Attachment
	Warnings    []string
}

// Attachment is a workspace file uploaded to the page under Filename.
type Attachment struct {
	Filename string
	Path     documentdiscovery.DocumentPath
}
