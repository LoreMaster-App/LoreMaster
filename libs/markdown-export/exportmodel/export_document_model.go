// Package exportmodel is the neutral shape of a document every export format renders from: a
// short list of block and inline kinds with no Markdown syntax left in it, so each format is
// written against this model and not against the parser.
package exportmodel

// BlockKind says what a Block is.
type BlockKind int

// The kinds of block.
const (
	HeadingBlock BlockKind = iota
	ParagraphBlock
	ListBlock
	QuoteBlock
	CodeBlock
	TableBlock
	RuleBlock
	HTMLBlock
)

// InlineKind says what an Inline is.
type InlineKind int

// The kinds of inline.
const (
	TextInline InlineKind = iota
	CodeInline
	EmphasisInline
	StrongInline
	StrikeInline
	LinkInline
	ImageInline
	SoftBreakInline
	HardBreakInline
	HTMLInline
)

// Align is a table column's alignment.
type Align int

// The alignments a column can have.
const (
	AlignDefault Align = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// Inline is a run of text or a span of formatting around other inlines.
type Inline struct {
	Kind InlineKind
	// Text is the literal text of a text, code or HTML inline, and the alt text of an image.
	Text string
	// URL is where a link or an image points.
	URL string
	// Title is a link's or an image's tooltip.
	Title    string
	Children []Inline
}

// Cell is one table cell.
type Cell []Inline

// Item is one list item. Checked is nil for an ordinary item and points at the state of a task.
type Item struct {
	Checked *bool
	Blocks  []Block
}

// Block is one block of a document. Which fields matter depends on Kind.
type Block struct {
	Kind BlockKind
	// Level is a heading's level, 1 to 6.
	Level int
	// Anchor is the fragment a link to a heading uses.
	Anchor  string
	Inlines []Inline
	// Ordered and Start describe a list.
	Ordered bool
	Start   int
	Items   []Item
	// Children are the blocks of a quote.
	Children []Block
	// Lang and Text are a code block's language and its verbatim text, or a raw HTML block's text.
	Lang string
	Text string
	// Header, Rows and Aligns are a table.
	Header []Cell
	Rows   [][]Cell
	Aligns []Align
}

// Document is a whole document: the title it is known by, and its blocks in order. The title
// heading is part of Blocks.
type Document struct {
	Title  string
	Blocks []Block
}
