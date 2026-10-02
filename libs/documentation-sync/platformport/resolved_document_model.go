package platformport

// Document is a page body in a platform-neutral form: Markdown mapped into it, with
// every link and image already resolved to a page title, an attachment or a URL. Each
// adapter renders it into its platform's format (Confluence: storage XHTML).
type Document struct {
	Blocks []Block
}

// Block is a block-level node.
type Block interface{ isBlock() }

// Inline is an inline node.
type Inline interface{ isInline() }

// Block nodes.
type (
	Paragraph struct{ Inlines []Inline }
	Heading   struct {
		Level   int
		Inlines []Inline
	}
	Blockquote struct{ Blocks []Block }
	List       struct {
		Ordered bool
		Start   int
		Items   []ListItem
	}
	ListItem struct{ Blocks []Block }
	TaskList struct{ Items []TaskItem }
	TaskItem struct {
		Done    bool
		Inlines []Inline
	}
	Table struct {
		Header []TableCell
		Rows   [][]TableCell
		Align  []Alignment
	}
	TableCell     struct{ Inlines []Inline }
	ThematicBreak struct{}
	CodeBlock     struct {
		Language string
		Code     string
	}
	// Diagram is diagram source (Mermaid) with, once rendered, its picture as an
	// attachment on the page.
	Diagram struct {
		Language string
		Source   string
		Image    *AttachmentRef
	}
)

// Alignment is a table column's alignment.
type Alignment int

// Column alignments.
const (
	AlignDefault Alignment = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// Inline nodes.
type (
	Text          struct{ Value string }
	Emphasis      struct{ Inlines []Inline }
	Strong        struct{ Inlines []Inline }
	Strikethrough struct{ Inlines []Inline }
	CodeSpan      struct{ Value string }
	HardBreak     struct{}
	Image         struct {
		Source ImageSource
		Alt    string
		Title  string
		Width  int
	}
	Link struct {
		Target  LinkTarget
		Inlines []Inline
	}
)

// ImageSource is an *AttachmentRef or a *URLRef.
type ImageSource interface{ isImageSource() }

// LinkTarget is a PageLink, an *AttachmentRef or a *URLRef.
type LinkTarget interface{ isLinkTarget() }

// PageLink points at another synced page by its final title; URL is set once known.
type PageLink struct {
	Title  string
	Anchor string
	URL    string
}

// AttachmentRef is a file attached to the page being written.
type AttachmentRef struct{ Filename string }

// URLRef is an address outside the platform.
type URLRef struct{ URL string }

func (Paragraph) isBlock()     {}
func (Heading) isBlock()       {}
func (Blockquote) isBlock()    {}
func (List) isBlock()          {}
func (TaskList) isBlock()      {}
func (Table) isBlock()         {}
func (ThematicBreak) isBlock() {}
func (CodeBlock) isBlock()     {}
func (Diagram) isBlock()       {}

func (Text) isInline()          {}
func (Emphasis) isInline()      {}
func (Strong) isInline()        {}
func (Strikethrough) isInline() {}
func (CodeSpan) isInline()      {}
func (HardBreak) isInline()     {}
func (Image) isInline()         {}
func (Link) isInline()          {}

func (*AttachmentRef) isImageSource() {}
func (*URLRef) isImageSource()        {}
func (PageLink) isLinkTarget()        {}
func (*AttachmentRef) isLinkTarget()  {}
func (*URLRef) isLinkTarget()         {}
