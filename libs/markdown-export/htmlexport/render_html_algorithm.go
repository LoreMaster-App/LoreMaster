// Package htmlexport renders a document as one self-contained HTML file: the stylesheet is
// inlined and local images are embedded, so the file can be mailed or opened anywhere.
package htmlexport

import (
	"encoding/base64"
	"html"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Options says where the document came from.
type Options struct {
	// BaseDir is the folder relative image paths are read from. Empty leaves images as links.
	BaseDir string
}

// Render writes the document as a complete HTML page.
func Render(document exportmodel.Document, options Options) string {
	var page strings.Builder
	page.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	page.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	page.WriteString("<title>" + html.EscapeString(document.Title) + "</title>\n<style>\n" + stylesheet + "</style>\n</head>\n<body>\n<main>\n")
	writer := writer{options: options}
	page.WriteString(writer.blocks(document.Blocks))
	page.WriteString("</main>\n</body>\n</html>\n")

	return page.String()
}

type writer struct {
	options Options
}

func (w writer) blocks(blocks []exportmodel.Block) string {
	var out strings.Builder
	for _, block := range blocks {
		out.WriteString(w.block(block))
	}

	return out.String()
}

func (w writer) block(block exportmodel.Block) string {
	switch block.Kind {
	case exportmodel.HeadingBlock:
		level := strconv.Itoa(min(max(block.Level, 1), 6))
		id := ""
		if block.Anchor != "" {
			id = " id=\"" + html.EscapeString(block.Anchor) + "\""
		}

		return "<h" + level + id + ">" + w.inlines(block.Inlines) + "</h" + level + ">\n"
	case exportmodel.ParagraphBlock:
		return "<p>" + w.inlines(block.Inlines) + "</p>\n"
	case exportmodel.ListBlock:
		return w.list(block)
	case exportmodel.QuoteBlock:
		return "<blockquote>\n" + w.blocks(block.Children) + "</blockquote>\n"
	case exportmodel.CodeBlock:
		class := ""
		if block.Lang != "" {
			class = " class=\"language-" + html.EscapeString(block.Lang) + "\""
		}

		return "<pre><code" + class + ">" + html.EscapeString(block.Text) + "</code></pre>\n"
	case exportmodel.TableBlock:
		return w.table(block)
	case exportmodel.RuleBlock:
		return "<hr>\n"
	case exportmodel.HTMLBlock:
		return block.Text + "\n"
	default:
		return ""
	}
}

func (w writer) list(block exportmodel.Block) string {
	tag, attributes := "ul", ""
	if block.Ordered {
		tag = "ol"
		if block.Start > 1 {
			attributes = " start=\"" + strconv.Itoa(block.Start) + "\""
		}
	}
	var out strings.Builder
	out.WriteString("<" + tag + attributes + ">\n")
	for _, item := range block.Items {
		out.WriteString("<li>")
		if item.Checked != nil {
			checked := ""
			if *item.Checked {
				checked = " checked"
			}
			out.WriteString("<input type=\"checkbox\" disabled" + checked + "> ")
		}
		out.WriteString(w.itemBlocks(item.Blocks))
		out.WriteString("</li>\n")
	}
	out.WriteString("</" + tag + ">\n")

	return out.String()
}

// itemBlocks leaves a lone paragraph unwrapped, as a tight list is written.
func (w writer) itemBlocks(blocks []exportmodel.Block) string {
	if len(blocks) == 1 && blocks[0].Kind == exportmodel.ParagraphBlock {
		return w.inlines(blocks[0].Inlines)
	}
	var out strings.Builder
	for _, block := range blocks {
		if block.Kind == exportmodel.ParagraphBlock {
			out.WriteString(w.inlines(block.Inlines))

			continue
		}
		out.WriteString("\n" + w.block(block))
	}

	return out.String()
}

func (w writer) table(block exportmodel.Block) string {
	var out strings.Builder
	out.WriteString("<table>\n")
	if len(block.Header) > 0 {
		out.WriteString("<thead>\n" + w.row("th", block.Header, block.Aligns) + "</thead>\n")
	}
	out.WriteString("<tbody>\n")
	for _, row := range block.Rows {
		out.WriteString(w.row("td", row, block.Aligns))
	}
	out.WriteString("</tbody>\n</table>\n")

	return out.String()
}

func (w writer) row(tag string, cells []exportmodel.Cell, aligns []exportmodel.Align) string {
	var out strings.Builder
	out.WriteString("<tr>")
	for column, cell := range cells {
		style := ""
		if column < len(aligns) {
			switch aligns[column] {
			case exportmodel.AlignLeft:
				style = " style=\"text-align:left\""
			case exportmodel.AlignCenter:
				style = " style=\"text-align:center\""
			case exportmodel.AlignRight:
				style = " style=\"text-align:right\""
			case exportmodel.AlignDefault:
			}
		}
		out.WriteString("<" + tag + style + ">" + w.inlines(cell) + "</" + tag + ">")
	}
	out.WriteString("</tr>\n")

	return out.String()
}

func (w writer) inlines(inlines []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range inlines {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(html.EscapeString(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString("<code>" + html.EscapeString(inline.Text) + "</code>")
		case exportmodel.EmphasisInline:
			out.WriteString("<em>" + w.inlines(inline.Children) + "</em>")
		case exportmodel.StrongInline:
			out.WriteString("<strong>" + w.inlines(inline.Children) + "</strong>")
		case exportmodel.StrikeInline:
			out.WriteString("<del>" + w.inlines(inline.Children) + "</del>")
		case exportmodel.LinkInline:
			out.WriteString("<a href=\"" + html.EscapeString(linkTarget(inline.URL)) + "\">" + w.inlines(inline.Children) + "</a>")
		case exportmodel.ImageInline:
			out.WriteString("<img src=\"" + html.EscapeString(w.imageSource(inline.URL)) + "\" alt=\"" + html.EscapeString(inline.Text) + "\">")
		case exportmodel.SoftBreakInline:
			out.WriteString("\n")
		case exportmodel.HardBreakInline:
			out.WriteString("<br>\n")
		case exportmodel.HTMLInline:
			out.WriteString(inline.Text)
		}
	}

	return out.String()
}

// linkTarget points a link to another Markdown file at the HTML file that file exports to.
func linkTarget(target string) string {
	path, fragment, hasFragment := strings.Cut(target, "#")
	if strings.Contains(path, "://") || strings.HasPrefix(path, "mailto:") {
		return target
	}
	for _, extension := range []string{".md", ".markdown"} {
		if strings.HasSuffix(path, extension) {
			path = strings.TrimSuffix(path, extension) + ".html"
		}
	}
	if hasFragment {
		return path + "#" + fragment
	}

	return path
}

// imageSource embeds a local image as a data URI, and leaves anything it cannot read as written.
func (w writer) imageSource(source string) string {
	if w.options.BaseDir == "" || strings.Contains(source, "://") || strings.HasPrefix(source, "data:") {
		return source
	}
	content, err := os.ReadFile(filepath.Join(w.options.BaseDir, filepath.FromSlash(source)))
	if err != nil {
		return source
	}
	kind := mime.TypeByExtension(strings.ToLower(filepath.Ext(source)))
	if kind == "" {
		return source
	}

	return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(content)
}
