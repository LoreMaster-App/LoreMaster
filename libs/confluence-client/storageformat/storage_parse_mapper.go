package storageformat

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// Parse reads Confluence storage format and reconstructs a Document — the reverse of Render,
// for the two-way sync pull (#160). Storage format is richer than the Document model, so Parse
// is conservative: it rebuilds the shapes Render emits and appends a human-readable flag for
// anything it does not recognise (an unknown macro, element or attribute), leaving that content
// out rather than guessing. A caller that sees flags should treat the conversion as lossy and
// not overwrite the user's file. The error is non-nil only when the input cannot be tokenised
// as XML at all.
func Parse(storage string) (Document, []string, error) {
	root, err := parseTree(storage)
	if err != nil {
		return Document{}, nil, err
	}
	p := &parser{}

	return Document{Blocks: p.blocks(root)}, p.flags, nil
}

type parser struct {
	flags []string
}

func (p *parser) flag(format string, a ...any) {
	p.flags = append(p.flags, fmt.Sprintf(format, a...))
}

// element is one parsed node with its children in document order; a child is text when its
// element is nil. Building this ordered tree first keeps interleaved text and elements in
// order, which struct unmarshalling loses for mixed content.
type element struct {
	name  xml.Name
	attrs []xml.Attr
	kids  []child
}

type child struct {
	text string
	el   *element
}

func parseTree(storage string) (*element, error) {
	dec := xml.NewDecoder(strings.NewReader("<lm-root>" + storage + "</lm-root>"))
	// Storage format is XHTML-ish: undeclared ac:/ri: prefixes (kept as the namespace) and
	// self-closing tags are fine, but non-strict mode tolerates the odd quirk a real page may
	// carry without failing the whole pull.
	dec.Strict = false
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if start, ok := tok.(xml.StartElement); ok {
			return readElement(dec, start)
		}
	}
}

func readElement(dec *xml.Decoder, start xml.StartElement) (*element, error) {
	e := &element{name: start.Name, attrs: start.Attr}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			kid, err := readElement(dec, t)
			if err != nil {
				return nil, err
			}
			e.kids = append(e.kids, child{el: kid})
		case xml.EndElement:
			return e, nil
		case xml.CharData:
			// Merge adjacent text so a run split across character references or CDATA sections
			// (for example an escaped "]]>") becomes one Text node.
			s := string(t)
			if n := len(e.kids); n > 0 && e.kids[n-1].el == nil {
				e.kids[n-1].text += s
			} else {
				e.kids = append(e.kids, child{text: s})
			}
		}
	}
}

// blocks reads an element's children as block content, grouping each run of inline nodes into a
// Paragraph so both a tight list item (<li>text) and a loose one (<li><p>text</p>) come back as
// an item holding paragraphs.
func (p *parser) blocks(e *element) []Block {
	var blocks []Block
	var pending []Inline
	flush := func() {
		if len(pending) > 0 {
			blocks = append(blocks, Paragraph{Inlines: pending})
			pending = nil
		}
	}
	for _, k := range e.kids {
		if k.el == nil {
			if strings.TrimSpace(k.text) == "" {
				continue
			}
			pending = append(pending, Text{Value: k.text})

			continue
		}
		if !isBlockElement(k.el) {
			p.inlineInto(k.el, &pending)

			continue
		}
		flush()
		block, ok := p.block(k.el)
		if !ok {
			continue
		}
		blocks = p.appendBlock(blocks, block)
	}
	flush()

	return blocks
}

// appendBlock adds block, folding an image-mode Mermaid's rendered picture back into it: Render
// writes that diagram as a <p> holding only the SVG attachment followed by a collapsed code
// macro, so when the macro arrives after such a paragraph the paragraph is the picture.
func (p *parser) appendBlock(blocks []Block, block Block) []Block {
	if diagram, ok := block.(Mermaid); ok {
		if n := len(blocks); n > 0 {
			if ref, only := imageOnlyParagraph(blocks[n-1]); only {
				diagram.Image = ref
				blocks = blocks[:n-1]
				block = diagram
			}
		}
	}

	return append(blocks, block)
}

func isBlockElement(e *element) bool {
	switch e.name.Space {
	case "":
		switch e.name.Local {
		case "p", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "ul", "ol", "table", "hr":
			return true
		}
	case "ac":
		switch e.name.Local {
		case "task-list", "structured-macro":
			return true
		}
	}

	return false
}

func (p *parser) block(e *element) (Block, bool) {
	if e.name.Space == "ac" {
		switch e.name.Local {
		case "task-list":
			return p.taskList(e), true
		case "structured-macro":
			return p.macro(e)
		}
		p.flag("an unknown macro <%s> was dropped", displayName(e))

		return nil, false
	}
	switch e.name.Local {
	case "p":
		return Paragraph{Inlines: p.inlines(e)}, true
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return Heading{Level: int(e.name.Local[1] - '0'), Inlines: p.inlines(e)}, true
	case "blockquote":
		return Blockquote{Blocks: p.blocks(e)}, true
	case "ul":
		return List{Items: p.listItems(e)}, true
	case "ol":
		return List{Ordered: true, Start: atoi(attr(e, "", "start")), Items: p.listItems(e)}, true
	case "hr":
		return ThematicBreak{}, true
	case "table":
		return p.table(e), true
	}
	p.flag("an unknown block <%s> was dropped", displayName(e))

	return nil, false
}

func (p *parser) listItems(e *element) []ListItem {
	var items []ListItem
	for _, k := range e.kids {
		if k.el != nil && k.el.name.Space == "" && k.el.name.Local == "li" {
			items = append(items, ListItem{Blocks: p.blocks(k.el)})
		}
	}

	return items
}

func (p *parser) taskList(e *element) TaskList {
	var items []TaskItem
	for _, k := range e.kids {
		if k.el != nil && k.el.name.Space == "ac" && k.el.name.Local == "task" {
			items = append(items, p.taskItem(k.el))
		}
	}

	return TaskList{Items: items}
}

func (p *parser) taskItem(e *element) TaskItem {
	item := TaskItem{}
	for _, k := range e.kids {
		if k.el == nil || k.el.name.Space != "ac" {
			continue
		}
		switch k.el.name.Local {
		case "task-status":
			item.Done = strings.TrimSpace(textOf(k.el)) == "complete"
		case "task-body":
			item.Inlines, item.Blocks = p.inlineLead(k.el)
		}
	}

	return item
}

// inlineLead reads a task body: its leading inline run becomes the item's text and any
// block-level children (a nested list, say) become its nested blocks, as Render writes them.
func (p *parser) inlineLead(e *element) ([]Inline, []Block) {
	var inlines []Inline
	var blocks []Block
	for _, k := range e.kids {
		if k.el == nil {
			if strings.TrimSpace(k.text) != "" || len(inlines) > 0 {
				inlines = append(inlines, Text{Value: k.text})
			}

			continue
		}
		if isBlockElement(k.el) {
			if b, ok := p.block(k.el); ok {
				blocks = p.appendBlock(blocks, b)
			}

			continue
		}
		p.inlineInto(k.el, &inlines)
	}

	return inlines, blocks
}

func (p *parser) macro(e *element) (Block, bool) {
	name := attr(e, "ac", "name")
	if name != "code" {
		p.flag("an unknown macro %q was dropped", name)

		return nil, false
	}
	var language, body string
	collapse, mermaidMarked := false, false
	for _, k := range e.kids {
		if k.el == nil || k.el.name.Space != "ac" {
			continue
		}
		switch k.el.name.Local {
		case "parameter":
			switch attr(k.el, "ac", "name") {
			case "language":
				language = textOf(k.el)
			case "collapse":
				collapse = strings.TrimSpace(textOf(k.el)) == "true"
			case "lore-master":
				mermaidMarked = strings.TrimSpace(textOf(k.el)) == "mermaid"
			}
		case "plain-text-body":
			body = textOf(k.el)
		}
	}
	// A code macro is a Mermaid diagram's source when it carries the lore-master marker. The
	// collapse flag is a fallback for pages synced before the marker existed, where only an
	// image-mode diagram's macro was collapsed; any other code macro is a plain code block.
	if mermaidMarked || collapse {
		return Mermaid{Source: body}, true
	}

	return CodeBlock{Language: language, Code: body}, true
}

func (p *parser) table(e *element) Block {
	var rows []*element
	var collect func(*element)
	collect = func(n *element) {
		for _, k := range n.kids {
			if k.el == nil {
				continue
			}
			switch k.el.name.Local {
			case "tr":
				rows = append(rows, k.el)
			case "tbody", "thead", "tfoot":
				collect(k.el)
			}
		}
	}
	collect(e)
	if len(rows) == 0 {
		return Table{}
	}
	table := Table{Align: p.alignOf(rows[0])}
	bodyStart := 0
	if headerCells, isHeader := p.rowCells(rows[0]); isHeader {
		table.Header = headerCells
		bodyStart = 1
	}
	for _, row := range rows[bodyStart:] {
		cells, _ := p.rowCells(row)
		table.Rows = append(table.Rows, cells)
	}

	return table
}

// rowCells returns a row's cells and whether the row is a header (holds any <th>).
func (p *parser) rowCells(tr *element) ([]TableCell, bool) {
	var cells []TableCell
	header := false
	for _, k := range tr.kids {
		if k.el == nil || (k.el.name.Local != "th" && k.el.name.Local != "td") {
			continue
		}
		if k.el.name.Local == "th" {
			header = true
		}
		cells = append(cells, TableCell{Inlines: p.inlines(k.el)})
	}

	return cells, header
}

func (p *parser) alignOf(tr *element) []Alignment {
	var align []Alignment
	for _, k := range tr.kids {
		if k.el == nil || (k.el.name.Local != "th" && k.el.name.Local != "td") {
			continue
		}
		align = append(align, parseAlign(attr(k.el, "", "style")))
	}

	return align
}

func parseAlign(style string) Alignment {
	switch {
	case strings.Contains(style, "left"):
		return AlignLeft
	case strings.Contains(style, "center"):
		return AlignCenter
	case strings.Contains(style, "right"):
		return AlignRight
	default:
		return AlignDefault
	}
}

func (p *parser) inlines(e *element) []Inline {
	var out []Inline
	for _, k := range e.kids {
		if k.el == nil {
			if k.text != "" {
				out = append(out, Text{Value: k.text})
			}

			continue
		}
		p.inlineInto(k.el, &out)
	}

	return out
}

func (p *parser) inlineInto(e *element, out *[]Inline) {
	if e.name.Space == "ac" {
		switch e.name.Local {
		case "image":
			if img, ok := p.image(e); ok {
				*out = append(*out, img)
			}

			return
		case "link":
			if link, ok := p.acLink(e); ok {
				*out = append(*out, link)
			}

			return
		}
		p.flag("an unknown inline macro <%s> was unwrapped", displayName(e))
		*out = append(*out, p.inlines(e)...)

		return
	}
	switch e.name.Local {
	case "em":
		*out = append(*out, Emphasis{Inlines: p.inlines(e)})
	case "strong":
		*out = append(*out, Strong{Inlines: p.inlines(e)})
	case "span":
		if strings.Contains(attr(e, "", "style"), "line-through") {
			*out = append(*out, Strikethrough{Inlines: p.inlines(e)})
		} else {
			p.flag("an unknown <span> was unwrapped")
			*out = append(*out, p.inlines(e)...)
		}
	case "code":
		*out = append(*out, CodeSpan{Value: textOf(e)})
	case "br":
		*out = append(*out, HardBreak{})
	case "a":
		*out = append(*out, Link{Target: &URLRef{URL: attr(e, "", "href")}, Inlines: p.inlines(e)})
	default:
		p.flag("an unknown inline <%s> was unwrapped", displayName(e))
		*out = append(*out, p.inlines(e)...)
	}
}

func (p *parser) image(e *element) (Image, bool) {
	img := Image{Alt: attr(e, "ac", "alt"), Title: attr(e, "ac", "title"), Width: atoi(attr(e, "ac", "width"))}
	for _, k := range e.kids {
		if k.el == nil || k.el.name.Space != "ri" {
			continue
		}
		switch k.el.name.Local {
		case "attachment":
			img.Source = &AttachmentRef{Filename: attr(k.el, "ri", "filename")}
		case "url":
			img.Source = &URLRef{URL: attr(k.el, "ri", "value")}
		}
	}
	if img.Source == nil {
		p.flag("an <ac:image> had no attachment or URL and was dropped")

		return Image{}, false
	}

	return img, true
}

func (p *parser) acLink(e *element) (Link, bool) {
	link := Link{}
	anchor := attr(e, "ac", "anchor")
	var body *element
	for _, k := range e.kids {
		if k.el == nil {
			continue
		}
		switch {
		case k.el.name.Space == "ri" && k.el.name.Local == "page":
			link.Target = PageLink{Title: attr(k.el, "ri", "content-title"), Anchor: anchor}
		case k.el.name.Space == "ri" && k.el.name.Local == "attachment":
			link.Target = &AttachmentRef{Filename: attr(k.el, "ri", "filename")}
		case k.el.name.Space == "ac" && k.el.name.Local == "link-body":
			body = k.el
		}
	}
	if link.Target == nil {
		p.flag("an <ac:link> had no page or attachment target and was dropped")

		return Link{}, false
	}
	if body != nil {
		link.Inlines = p.inlines(body)
	}

	return link, true
}

// imageOnlyParagraph reports a paragraph holding nothing but one attachment image, the shape
// Render uses for an image-mode diagram's rendered picture.
func imageOnlyParagraph(b Block) (*AttachmentRef, bool) {
	para, ok := b.(Paragraph)
	if !ok || len(para.Inlines) != 1 {
		return nil, false
	}
	img, ok := para.Inlines[0].(Image)
	if !ok {
		return nil, false
	}
	ref, ok := img.Source.(*AttachmentRef)

	return ref, ok
}

func attr(e *element, space, local string) string {
	for _, a := range e.attrs {
		if a.Name.Local == local && (space == "" || a.Name.Space == space) {
			return a.Value
		}
	}

	return ""
}

func textOf(e *element) string {
	var b strings.Builder
	for _, k := range e.kids {
		if k.el == nil {
			b.WriteString(k.text)
		} else {
			b.WriteString(textOf(k.el))
		}
	}

	return b.String()
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))

	return n
}

func displayName(e *element) string {
	if e.name.Space != "" {
		return e.name.Space + ":" + e.name.Local
	}

	return e.name.Local
}
