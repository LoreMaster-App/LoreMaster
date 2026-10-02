package documentparsing

import (
	"github.com/yuin/goldmark/ast"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// MarkdownDocument is one parsed Markdown file.
type MarkdownDocument struct {
	Path documentdiscovery.DocumentPath
	// Title is the first top-level H1 as plain text, or the file name when there is none.
	Title string
	// TitleFromHeading says whether Title came from an H1 rather than the file name.
	TitleFromHeading bool
	// Order is the file name's numeric prefix ("01-intro.md" is 1); Ordered says whether
	// there was one. Siblings sort by it before their titles.
	Order   int
	Ordered bool
	// AST is the goldmark syntax tree. Its segments index into Body.
	AST ast.Node
	// Body is the file without its byte-order mark and lore-master annotation.
	Body []byte
	// Annotation is the file's lore-master annotation, or nil.
	Annotation *syncannotation.Annotation
	Layout     syncannotation.Layout
	Warnings   []string
}
