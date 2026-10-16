// Package odtexport renders a document as an OpenDocument text file (.odt), the format of
// LibreOffice and of public bodies that require an open standard. It writes the zipped XML directly.
package odtexport

import (
	"fmt"
	"html"
	"path"
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
	"lore-master/libs/markdown-export/imageloading"
)

// pixelsPerCentimetre converts pixels at 96 dpi to centimetres.
const pixelsPerCentimetre = 37.8

// maxImageWidth is the text width of an A4 page with 2.5 cm margins, in centimetres.
const maxImageWidth = 16.0

// picture is an image stored in the package.
type picture struct {
	name string
	mime string
	data []byte
}

// content writes the body of content.xml and collects the pictures it refers to.
type content struct {
	baseDir  string
	out      strings.Builder
	pictures []picture
	tables   int
}

func (c *content) blocks(list []exportmodel.Block) {
	for _, current := range list {
		c.block(current)
	}
}

func (c *content) block(current exportmodel.Block) {
	switch current.Kind {
	case exportmodel.HeadingBlock:
		level := strconv.Itoa(min(max(current.Level, 1), 6))
		c.out.WriteString(`<text:h text:style-name="Heading_20_` + level + `" text:outline-level="` + level + `">`)
		if current.Anchor != "" {
			c.out.WriteString(`<text:bookmark text:name="` + escape(current.Anchor) + `"/>`)
		}
		c.out.WriteString(c.spans(current.Inlines) + "</text:h>")
	case exportmodel.ParagraphBlock:
		c.out.WriteString(`<text:p text:style-name="Text_20_body">` + c.spans(current.Inlines) + "</text:p>")
	case exportmodel.ListBlock:
		c.list(current)
	case exportmodel.QuoteBlock:
		c.quote(current.Children)
	case exportmodel.CodeBlock:
		c.out.WriteString(`<text:p text:style-name="Preformatted_20_Text">` + preformatted(current.Text) + "</text:p>")
	case exportmodel.TableBlock:
		c.table(current)
	case exportmodel.RuleBlock:
		c.out.WriteString(`<text:p text:style-name="Horizontal_20_Line"/>`)
	case exportmodel.HTMLBlock:
		c.out.WriteString(`<text:p text:style-name="Preformatted_20_Text">` + preformatted(current.Text) + "</text:p>")
	}
}

func (c *content) quote(children []exportmodel.Block) {
	for _, child := range children {
		switch child.Kind {
		case exportmodel.ParagraphBlock, exportmodel.HeadingBlock:
			c.out.WriteString(`<text:p text:style-name="Quotations">` + c.spans(child.Inlines) + "</text:p>")
		case exportmodel.QuoteBlock:
			c.quote(child.Children)
		default:
			c.block(child)
		}
	}
}

// list writes a list in the bullet or number style. A nested list sits inside the item it belongs
// to and inherits the style, with the level taken from how deep it is.
func (c *content) list(current exportmodel.Block) {
	style := "L_bullet"
	if current.Ordered {
		style = "L_number"
	}
	c.out.WriteString(`<text:list text:style-name="` + style + `">`)
	for position, item := range current.Items {
		c.out.WriteString("<text:list-item")
		if current.Ordered && position == 0 && current.Start > 1 {
			c.out.WriteString(` text:start-value="` + strconv.Itoa(current.Start) + `"`)
		}
		c.out.WriteString(">")
		c.item(item)
		c.out.WriteString("</text:list-item>")
	}
	c.out.WriteString("</text:list>")
}

func (c *content) item(item exportmodel.Item) {
	marker := ""
	if item.Checked != nil {
		marker = "☐ "
		if *item.Checked {
			marker = "☒ "
		}
	}
	first := true
	for _, inner := range item.Blocks {
		if inner.Kind == exportmodel.ParagraphBlock {
			prefix := ""
			if first {
				prefix = marker
			}
			c.out.WriteString(`<text:p text:style-name="List_20_Contents">` + prefix + c.spans(inner.Inlines) + "</text:p>")
		} else {
			c.block(inner)
		}
		first = false
	}
	if first {
		c.out.WriteString(`<text:p text:style-name="List_20_Contents">` + marker + "</text:p>")
	}
}

func (c *content) table(current exportmodel.Block) {
	columns := len(current.Header)
	for _, row := range current.Rows {
		columns = max(columns, len(row))
	}
	if columns == 0 {
		return
	}
	c.tables++
	c.out.WriteString(`<table:table table:name="Table` + strconv.Itoa(c.tables) + `" table:style-name="Grid">`)
	c.out.WriteString(`<table:table-column table:number-columns-repeated="` + strconv.Itoa(columns) + `"/>`)
	if len(current.Header) > 0 {
		c.out.WriteString("<table:table-header-rows>")
		c.row(current.Header, current.Aligns, columns, true)
		c.out.WriteString("</table:table-header-rows>")
	}
	for _, row := range current.Rows {
		c.row(row, current.Aligns, columns, false)
	}
	c.out.WriteString("</table:table>")
}

func (c *content) row(cells []exportmodel.Cell, aligns []exportmodel.Align, columns int, header bool) {
	c.out.WriteString("<table:table-row>")
	for column := range columns {
		var cell exportmodel.Cell
		if column < len(cells) {
			cell = cells[column]
		}
		style := "Table_20_Contents"
		switch {
		case header:
			style = "Table_20_Heading"
		case column < len(aligns) && aligns[column] == exportmodel.AlignCenter:
			style = "Cell_20_Centre"
		case column < len(aligns) && aligns[column] == exportmodel.AlignRight:
			style = "Cell_20_Right"
		}
		c.out.WriteString(`<table:table-cell table:style-name="Cell" office:value-type="string"><text:p text:style-name="` + style + `">` + c.spans(cell) + "</text:p></table:table-cell>")
	}
	c.out.WriteString("</table:table-row>")
}

func (c *content) spans(inlines []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range inlines {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString(`<text:span text:style-name="Source_20_Text">` + escape(inline.Text) + "</text:span>")
		case exportmodel.EmphasisInline:
			out.WriteString(`<text:span text:style-name="Emphasis">` + c.spans(inline.Children) + "</text:span>")
		case exportmodel.StrongInline:
			out.WriteString(`<text:span text:style-name="Strong_20_Emphasis">` + c.spans(inline.Children) + "</text:span>")
		case exportmodel.StrikeInline:
			out.WriteString(`<text:span text:style-name="Strikethrough">` + c.spans(inline.Children) + "</text:span>")
		case exportmodel.LinkInline:
			out.WriteString(c.link(inline))
		case exportmodel.ImageInline:
			out.WriteString(c.image(inline))
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("<text:line-break/>")
		case exportmodel.HTMLInline:
		}
	}

	return out.String()
}

// link writes a text:a. A link to another Markdown file points at the OpenDocument file it exports to.
func (c *content) link(inline exportmodel.Inline) string {
	label := c.spans(inline.Children)
	if inline.URL == "" {
		return label
	}
	target := inline.URL
	if !strings.Contains(target, "://") && !strings.HasPrefix(target, "mailto:") && !strings.HasPrefix(target, "#") {
		file, fragment, hasFragment := strings.Cut(target, "#")
		for _, extension := range []string{".markdown", ".md"} {
			if strings.HasSuffix(file, extension) {
				file = strings.TrimSuffix(file, extension)

				break
			}
		}
		target = file + ".odt"
		if hasFragment {
			target += "#" + fragment
		}
	}

	return `<text:a xlink:type="simple" xlink:href="` + escape(target) + `" text:style-name="Internet_20_link">` + label + "</text:a>"
}

// image places a local picture in the text, or writes its alt text when it cannot be embedded.
func (c *content) image(inline exportmodel.Inline) string {
	loaded, found := imageloading.LoadImage(c.baseDir, inline.URL)
	if !found {
		label := inline.Text
		if label == "" {
			label = path.Base(inline.URL)
		}

		return escape(label)
	}
	name := fmt.Sprintf("image%d.%s", len(c.pictures)+1, loaded.Extension)
	c.pictures = append(c.pictures, picture{name: name, mime: loaded.MIME, data: loaded.Data})

	width := float64(loaded.Width) / pixelsPerCentimetre
	height := float64(loaded.Height) / pixelsPerCentimetre
	if width > maxImageWidth {
		height = height * maxImageWidth / width
		width = maxImageWidth
	}

	return fmt.Sprintf(`<draw:frame draw:name="Image%d" text:anchor-type="as-char" svg:width="%.2fcm" svg:height="%.2fcm">`+
		`<draw:image xlink:href="Pictures/%s" xlink:type="simple" xlink:show="embed" xlink:actuate="onLoad"/></draw:frame>`,
		len(c.pictures), width, height, name)
}

// preformatted writes code the way it was written: line breaks, tabs and runs of spaces are kept,
// which OpenDocument would otherwise collapse.
func preformatted(text string) string {
	var out strings.Builder
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			out.WriteString("<text:line-break/>")
		}
		line = strings.TrimRight(line, "\r")
		spaces := 0
		for _, character := range line {
			if character == ' ' {
				spaces++

				continue
			}
			out.WriteString(spacesRun(spaces))
			spaces = 0
			if character == '\t' {
				out.WriteString("<text:tab/>")

				continue
			}
			out.WriteString(escape(string(character)))
		}
		out.WriteString(spacesRun(spaces))
	}

	return out.String()
}

func spacesRun(count int) string {
	switch count {
	case 0:
		return ""
	case 1:
		return " "
	default:
		return `<text:s text:c="` + strconv.Itoa(count) + `"/>`
	}
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
