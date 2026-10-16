// Package docxexport renders a document as a Word file (.docx): zipped WordprocessingML written
// directly, so it needs no Office installation and no dependency.
package docxexport

import (
	"fmt"
	"html"
	"path"
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
	"lore-master/libs/markdown-export/imageloading"
)

// emuPerPixel converts pixels at 96 dpi to the English metric units Word measures drawings in.
const emuPerPixel = 9525

// maxImageWidth is six inches in EMU: the text width of a default page.
const maxImageWidth = 6 * 914400

// relationship is one target the document part refers to by id.
type relationship struct {
	id, kind, target string
	external         bool
}

// orderedList is where an ordered list sits and the number it starts at.
type orderedList struct {
	level, start int
}

// media is an image file stored in the package.
type media struct {
	name string
	data []byte
}

// format is the inline formatting in force while writing runs.
type format struct {
	bold, italic, strike, code, link bool
}

// body writes the document part, collecting what the other parts need: relationships, images and
// numbering instances.
type body struct {
	baseDir       string
	out           strings.Builder
	relationships []relationship
	media         []media
	// ordered holds every ordered list met; its numbering instance id is its position here plus
	// orderedBase.
	ordered []orderedList
	bookmarks     int
}

// The numbering instance of every bullet list, and the first instance an ordered list gets.
const (
	bulletNumID = 1
	orderedBase = 2
)

func (b *body) blocks(list []exportmodel.Block, depth int) {
	for _, current := range list {
		b.block(current, depth)
	}
}

func (b *body) block(current exportmodel.Block, depth int) {
	switch current.Kind {
	case exportmodel.HeadingBlock:
		b.heading(current)
	case exportmodel.ParagraphBlock:
		b.paragraph("", "", b.runs(current.Inlines, format{}))
	case exportmodel.ListBlock:
		b.list(current, depth)
	case exportmodel.QuoteBlock:
		b.quote(current.Children)
	case exportmodel.CodeBlock:
		b.code(current.Text)
	case exportmodel.TableBlock:
		b.table(current)
	case exportmodel.RuleBlock:
		b.out.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="999999"/></w:pBdr></w:pPr></w:p>`)
	case exportmodel.HTMLBlock:
		b.code(current.Text)
	}
}

// paragraph writes one paragraph with an optional style and extra paragraph properties.
func (b *body) paragraph(style, properties, content string) {
	b.out.WriteString("<w:p>")
	if style != "" || properties != "" {
		b.out.WriteString("<w:pPr>")
		if style != "" {
			b.out.WriteString(`<w:pStyle w:val="` + style + `"/>`)
		}
		b.out.WriteString(properties + "</w:pPr>")
	}
	b.out.WriteString(content + "</w:p>")
}

func (b *body) heading(current exportmodel.Block) {
	content := b.runs(current.Inlines, format{})
	if current.Anchor != "" {
		b.bookmarks++
		id := strconv.Itoa(b.bookmarks)
		content = `<w:bookmarkStart w:id="` + id + `" w:name="` + escape(current.Anchor) + `"/>` + content + `<w:bookmarkEnd w:id="` + id + `"/>`
	}
	b.paragraph("Heading"+strconv.Itoa(min(max(current.Level, 1), 6)), "", content)
}

func (b *body) quote(children []exportmodel.Block) {
	for _, child := range children {
		switch child.Kind {
		case exportmodel.ParagraphBlock, exportmodel.HeadingBlock:
			b.paragraph("Quote", "", b.runs(child.Inlines, format{}))
		case exportmodel.QuoteBlock:
			b.quote(child.Children)
		default:
			b.block(child, 0)
		}
	}
}

// code writes a code block as one paragraph in the monospace style, a line break between lines.
func (b *body) code(text string) {
	lines := strings.Split(text, "\n")
	var content strings.Builder
	for i, line := range lines {
		if i > 0 {
			content.WriteString("<w:r><w:br/></w:r>")
		}
		content.WriteString(run(strings.TrimRight(line, "\r"), format{}))
	}
	b.paragraph("Code", "", content.String())
}

func (b *body) list(current exportmodel.Block, depth int) {
	level := min(depth, 8)
	numID := bulletNumID
	if current.Ordered {
		numID = orderedBase + len(b.ordered)
		b.ordered = append(b.ordered, orderedList{level: level, start: max(current.Start, 1)})
	}
	properties := `<w:numPr><w:ilvl w:val="` + strconv.Itoa(level) + `"/><w:numId w:val="` + strconv.Itoa(numID) + `"/></w:numPr>`
	for _, item := range current.Items {
		b.item(item, properties, depth)
	}
}

func (b *body) item(item exportmodel.Item, properties string, depth int) {
	marker := ""
	if item.Checked != nil {
		marker = run("☐ ", format{})
		if *item.Checked {
			marker = run("☒ ", format{})
		}
	}
	first := true
	for _, inner := range item.Blocks {
		switch inner.Kind {
		case exportmodel.ParagraphBlock:
			content := b.runs(inner.Inlines, format{})
			if first {
				content = marker + content
				b.paragraph("ListParagraph", properties, content)
			} else {
				b.paragraph("ListParagraph", `<w:ind w:left="`+strconv.Itoa(720*(depth+1))+`"/>`, content)
			}
		case exportmodel.ListBlock:
			b.list(inner, depth+1)
		default:
			b.block(inner, depth+1)
		}
		first = false
	}
	if first {
		b.paragraph("ListParagraph", properties, marker)
	}
}

func (b *body) table(current exportmodel.Block) {
	columns := len(current.Header)
	for _, row := range current.Rows {
		columns = max(columns, len(row))
	}
	if columns == 0 {
		return
	}
	b.out.WriteString(`<w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="0" w:type="auto"/></w:tblPr><w:tblGrid>`)
	b.out.WriteString(strings.Repeat(`<w:gridCol w:w="`+strconv.Itoa(9000/columns)+`"/>`, columns))
	b.out.WriteString("</w:tblGrid>")
	if len(current.Header) > 0 {
		b.row(current.Header, current.Aligns, columns, true)
	}
	for _, row := range current.Rows {
		b.row(row, current.Aligns, columns, false)
	}
	// Word wants a paragraph after a table, and it keeps the next block off the table's edge.
	b.out.WriteString("</w:tbl><w:p/>")
}

func (b *body) row(cells []exportmodel.Cell, aligns []exportmodel.Align, columns int, header bool) {
	b.out.WriteString("<w:tr>")
	if header {
		b.out.WriteString("<w:trPr><w:tblHeader/></w:trPr>")
	}
	for column := range columns {
		var cell exportmodel.Cell
		if column < len(cells) {
			cell = cells[column]
		}
		properties := ""
		if column < len(aligns) {
			properties = alignment(aligns[column])
		}
		b.out.WriteString(`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr>`)
		b.paragraph("", properties, b.runs(cell, format{bold: header}))
		b.out.WriteString("</w:tc>")
	}
	b.out.WriteString("</w:tr>")
}

func alignment(align exportmodel.Align) string {
	switch align {
	case exportmodel.AlignCenter:
		return `<w:jc w:val="center"/>`
	case exportmodel.AlignRight:
		return `<w:jc w:val="right"/>`
	case exportmodel.AlignLeft, exportmodel.AlignDefault:
		return ""
	default:
		return ""
	}
}

func (b *body) runs(inlines []exportmodel.Inline, current format) string {
	var out strings.Builder
	for _, inline := range inlines {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(run(inline.Text, current))
		case exportmodel.CodeInline:
			withCode := current
			withCode.code = true
			out.WriteString(run(inline.Text, withCode))
		case exportmodel.EmphasisInline:
			out.WriteString(b.runs(inline.Children, format{bold: current.bold, italic: true, strike: current.strike, code: current.code, link: current.link}))
		case exportmodel.StrongInline:
			out.WriteString(b.runs(inline.Children, format{bold: true, italic: current.italic, strike: current.strike, code: current.code, link: current.link}))
		case exportmodel.StrikeInline:
			out.WriteString(b.runs(inline.Children, format{bold: current.bold, italic: current.italic, strike: true, code: current.code, link: current.link}))
		case exportmodel.LinkInline:
			out.WriteString(b.link(inline, current))
		case exportmodel.ImageInline:
			out.WriteString(b.image(inline, current))
		case exportmodel.SoftBreakInline:
			out.WriteString(run(" ", current))
		case exportmodel.HardBreakInline:
			out.WriteString("<w:r><w:br/></w:r>")
		case exportmodel.HTMLInline:
		}
	}

	return out.String()
}

// link wraps a link's runs in a hyperlink: to a heading of the document by its bookmark, and to
// anything else through a relationship. A link to another Markdown file points at the Word file
// that file exports to.
func (b *body) link(inline exportmodel.Inline, current format) string {
	inner := current
	inner.link = true
	content := b.runs(inline.Children, inner)
	switch {
	case inline.URL == "":
		return content
	case strings.HasPrefix(inline.URL, "#"):
		return `<w:hyperlink w:anchor="` + escape(strings.TrimPrefix(inline.URL, "#")) + `" w:history="1">` + content + "</w:hyperlink>"
	}
	target := inline.URL
	if !strings.Contains(target, "://") && !strings.HasPrefix(target, "mailto:") {
		file, fragment, hasFragment := strings.Cut(target, "#")
		for _, extension := range []string{".markdown", ".md"} {
			if strings.HasSuffix(file, extension) {
				file = strings.TrimSuffix(file, extension)

				break
			}
		}
		target = file + ".docx"
		if hasFragment {
			target += "#" + fragment
		}
	}
	id := b.relate("http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink", target, true)

	return `<w:hyperlink r:id="` + id + `" w:history="1">` + content + "</w:hyperlink>"
}

// relate records a relationship and returns the id the document part refers to it by.
func (b *body) relate(kind, target string, external bool) string {
	id := "rId" + strconv.Itoa(len(b.relationships)+10)
	b.relationships = append(b.relationships, relationship{id: id, kind: kind, target: target, external: external})

	return id
}

// image embeds a local picture inline, or falls back to its alt text when it cannot be embedded.
func (b *body) image(inline exportmodel.Inline, current format) string {
	loaded, found := imageloading.LoadImage(b.baseDir, inline.URL)
	if !found {
		label := inline.Text
		if label == "" {
			label = path.Base(inline.URL)
		}

		return run(label, current)
	}
	name := fmt.Sprintf("image%d.%s", len(b.media)+1, loaded.Extension)
	b.media = append(b.media, media{name: name, data: loaded.Data})
	id := b.relate("http://schemas.openxmlformats.org/officeDocument/2006/relationships/image", "media/"+name, false)

	width, height := loaded.Width*emuPerPixel, loaded.Height*emuPerPixel
	if width > maxImageWidth {
		height = height * maxImageWidth / width
		width = maxImageWidth
	}
	number := strconv.Itoa(len(b.media))
	size := `cx="` + strconv.Itoa(width) + `" cy="` + strconv.Itoa(height) + `"`

	return `<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent ` + size + `/>` +
		`<wp:docPr id="` + number + `" name="Picture ` + number + `" descr="` + escape(inline.Text) + `"/>` +
		`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="` + number + `" name="` + name + `"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="` + id + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext ` + size + `/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic>` +
		`</a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
}

// run writes a piece of text with the formatting in force.
func run(text string, current format) string {
	if text == "" {
		return ""
	}
	var properties strings.Builder
	switch {
	case current.code:
		properties.WriteString(`<w:rStyle w:val="CodeChar"/>`)
	case current.link:
		properties.WriteString(`<w:rStyle w:val="Hyperlink"/>`)
	}
	if current.bold {
		properties.WriteString("<w:b/>")
	}
	if current.italic {
		properties.WriteString("<w:i/>")
	}
	if current.strike {
		properties.WriteString("<w:strike/>")
	}
	if properties.Len() > 0 {
		return `<w:r><w:rPr>` + properties.String() + `</w:rPr><w:t xml:space="preserve">` + escape(text) + `</w:t></w:r>`
	}

	return `<w:r><w:t xml:space="preserve">` + escape(text) + `</w:t></w:r>`
}

// escape makes text safe for XML: markup characters are escaped and the control characters XML
// forbids are dropped.
func escape(text string) string {
	cleaned := strings.Map(func(character rune) rune {
		if character < 0x20 && character != '\t' && character != '\n' && character != '\r' {
			return -1
		}

		return character
	}, text)

	return html.EscapeString(cleaned)
}
