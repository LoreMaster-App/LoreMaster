package storageformat

// Document is a page body to render.
type Document struct {
	Blocks []Block
}

// Block is a block-level node.
type Block interface{ isBlock() }

// Inline is an inline node.
type Inline interface{ isInline() }

// Paragraph is a run of inline content.
type Paragraph struct{ Inlines []Inline }

// Heading is h1 to h6.
type Heading struct {
	Level   int
	Inlines []Inline
}

// Blockquote holds blocks.
type Blockquote struct{ Blocks []Block }

// List is a bulleted or numbered list. Start is the first number of an ordered list;
// 0 and 1 both mean "from 1".
type List struct {
	Ordered bool
	Start   int
	Items   []ListItem
}

// ListItem holds blocks; a single paragraph renders inline in the item.
type ListItem struct{ Blocks []Block }

// TaskList is a list of checkboxes.
type TaskList struct{ Items []TaskItem }

// TaskItem is one checkbox and its text.
type TaskItem struct {
	Done    bool
	Inlines []Inline
}

// Alignment is a table column's alignment.
type Alignment int

// Column alignments.
const (
	AlignDefault Alignment = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// Table is a header row and body rows. Align has one entry per column (or fewer).
type Table struct {
	Header []TableCell
	Rows   [][]TableCell
	Align  []Alignment
}

// TableCell is one cell's inline content.
type TableCell struct{ Inlines []Inline }

// ThematicBreak is a horizontal rule.
type ThematicBreak struct{}

// Text is literal text; the renderer escapes it.
type Text struct{ Value string }

// Emphasis is italic text.
type Emphasis struct{ Inlines []Inline }

// Strong is bold text.
type Strong struct{ Inlines []Inline }

// Strikethrough is struck-out text.
type Strikethrough struct{ Inlines []Inline }

// CodeSpan is inline code.
type CodeSpan struct{ Value string }

// HardBreak is a line break inside a block.
type HardBreak struct{}

// CodeBlock is fenced or indented code. Language is the fence's info string.
type CodeBlock struct {
	Language string
	Code     string
}

// Mermaid is a Mermaid diagram. Image is its rendered picture, uploaded as an
// attachment; nil when none was rendered.
type Mermaid struct {
	Source string
	Image  *AttachmentRef
}

// Image is a picture; Source is an *AttachmentRef or a *URLRef. Width is in pixels, 0
// for the natural size.
type Image struct {
	Source ImageSource
	Alt    string
	Title  string
	Width  int
}

// Link is a hyperlink; Target is a PageLink, an *AttachmentRef or a *URLRef.
type Link struct {
	Target  LinkTarget
	Inlines []Inline
}

// ImageSource is where an image comes from.
type ImageSource interface{ isImageSource() }

// LinkTarget is what a link points at.
type LinkTarget interface{ isLinkTarget() }

// PageLink is a link to another synced page, by its final (prefixed) title, which
// Confluence resolves when the page is shown, so links between pages created in the
// same sync, cycles included, need no second pass. Anchor is a heading anchor on that
// page. URL is the page's address, used only in LinkByURL mode.
type PageLink struct {
	Title  string
	Anchor string
	URL    string
}

// AttachmentRef is a file attached to the page being rendered.
type AttachmentRef struct{ Filename string }

// URLRef is an address outside Confluence.
type URLRef struct{ URL string }

func (*AttachmentRef) isImageSource() {}
func (*URLRef) isImageSource()        {}
func (PageLink) isLinkTarget()        {}
func (*AttachmentRef) isLinkTarget()  {}
func (*URLRef) isLinkTarget()         {}

func (CodeBlock) isBlock() {}
func (Mermaid) isBlock()   {}
func (Image) isInline()    {}
func (Link) isInline()     {}

func (Paragraph) isBlock()     {}
func (Heading) isBlock()       {}
func (Blockquote) isBlock()    {}
func (List) isBlock()          {}
func (TaskList) isBlock()      {}
func (Table) isBlock()         {}
func (ThematicBreak) isBlock() {}

func (Text) isInline()          {}
func (Emphasis) isInline()      {}
func (Strong) isInline()        {}
func (Strikethrough) isInline() {}
func (CodeSpan) isInline()      {}
func (HardBreak) isInline()     {}
