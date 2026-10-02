package documentparsing

import (
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM, &frontmatter.Extender{}))

// ParseDocument parses one file's content: the annotation is split off, front matter is
// recognised and kept out of the tree, and the rest is parsed as GitHub Flavored
// Markdown (tables, task lists, strikethrough, autolinks).
func ParseDocument(documentPath documentdiscovery.DocumentPath, content []byte) (MarkdownDocument, error) {
	split, err := syncannotation.Read(content)
	if err != nil {
		return MarkdownDocument{}, fmt.Errorf("%s: %w", documentPath, err)
	}
	tree := markdown.Parser().Parse(text.NewReader(split.Body))

	document := MarkdownDocument{
		Path:       documentPath,
		AST:        tree,
		Body:       split.Body,
		Annotation: split.Annotation,
		Layout:     split.Layout,
		Warnings:   split.Warnings,
	}
	fallback, order, ordered := fileNameTitle(string(documentPath))
	document.Order, document.Ordered = order, ordered
	document.Title, document.TitleHeading = headingTitle(tree, split.Body)
	document.TitleFromHeading = document.TitleHeading != nil
	if !document.TitleFromHeading {
		document.Title = fallback
	}

	return document, nil
}
