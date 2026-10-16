// Package mediawikiexport renders a document as MediaWiki markup, the syntax of Wikipedia and of
// every wiki that runs MediaWiki.
package mediawikiexport

import (
	"path"
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document as wikitext.
func Render(document exportmodel.Document) string {
	return strings.Join(blocks(document.Blocks), "\n\n") + "\n"
}

func blocks(list []exportmodel.Block) []string {
	rendered := make([]string, 0, len(list))
	for _, current := range list {
		if text := block(current); text != "" {
			rendered = append(rendered, text)
		}
	}

	return rendered
}

func block(current exportmodel.Block) string {
	switch current.Kind {
	case exportmodel.HeadingBlock:
		marks := strings.Repeat("=", min(max(current.Level, 1), 6))

		return marks + " " + inlines(current.Inlines) + " " + marks
	case exportmodel.ParagraphBlock:
		return startOfLine(inlines(current.Inlines))
	case exportmodel.ListBlock:
		return list(current, "")
	case exportmodel.QuoteBlock:
		return "<blockquote>\n" + strings.Join(blocks(current.Children), "\n\n") + "\n</blockquote>"
	case exportmodel.CodeBlock:
		return code(current)
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "----"
	case exportmodel.HTMLBlock:
		return current.Text
	default:
		return ""
	}
}

func code(current exportmodel.Block) string {
	if current.Lang == "" {
		return "<pre>\n" + escapeHTML(current.Text) + "\n</pre>"
	}

	return "<syntaxhighlight lang=\"" + current.Lang + "\">\n" + current.Text + "\n</syntaxhighlight>"
}

func list(current exportmodel.Block, parent string) string {
	marker := "*"
	if current.Ordered {
		marker = "#"
	}
	prefix := parent + marker
	var lines []string
	for _, item := range current.Items {
		box := ""
		if item.Checked != nil {
			box = "&#9744; "
			if *item.Checked {
				box = "&#9745; "
			}
		}
		first := true
		for _, inner := range item.Blocks {
			switch inner.Kind {
			case exportmodel.ListBlock:
				lines = append(lines, list(inner, prefix))
			case exportmodel.ParagraphBlock:
				text := inlines(inner.Inlines)
				if first {
					text = box + text
				}
				lines = append(lines, prefix+" "+text)
			default:
				lines = append(lines, prefix+" "+strings.ReplaceAll(block(inner), "\n", " "))
			}
			first = false
		}
	}

	return strings.Join(lines, "\n")
}

func table(current exportmodel.Block) string {
	lines := []string{"{| class=\"wikitable\""}
	if len(current.Header) > 0 {
		lines = append(lines, "|-")
		for column, cell := range current.Header {
			lines = append(lines, "!"+cellStyle(current.Aligns, column)+inlines(cell))
		}
	}
	for _, row := range current.Rows {
		lines = append(lines, "|-")
		for column, cell := range row {
			lines = append(lines, "|"+cellStyle(current.Aligns, column)+inlines(cell))
		}
	}

	return strings.Join(append(lines, "|}"), "\n")
}

// cellStyle is the alignment attribute of a column's cells, ending in the "|" that separates
// attributes from content.
func cellStyle(aligns []exportmodel.Align, column int) string {
	if column >= len(aligns) {
		return " "
	}
	switch aligns[column] {
	case exportmodel.AlignLeft:
		return " style=\"text-align:left\" | "
	case exportmodel.AlignCenter:
		return " style=\"text-align:center\" | "
	case exportmodel.AlignRight:
		return " style=\"text-align:right\" | "
	default:
		return " "
	}
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString("<code>" + escapeHTML(inline.Text) + "</code>")
		case exportmodel.EmphasisInline:
			out.WriteString("''" + inlines(inline.Children) + "''")
		case exportmodel.StrongInline:
			out.WriteString("'''" + inlines(inline.Children) + "'''")
		case exportmodel.StrikeInline:
			out.WriteString("<s>" + inlines(inline.Children) + "</s>")
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString(image(inline))
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("<br />\n")
		case exportmodel.HTMLInline:
			out.WriteString(inline.Text)
		}
	}

	return out.String()
}

// link writes an external address as [url text] and a link to another Markdown file or to a
// heading as a wiki link, because a wiki names its pages and not its files.
func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if strings.Contains(inline.URL, "://") || strings.HasPrefix(inline.URL, "mailto:") {
		return "[" + inline.URL + " " + label + "]"
	}
	file, fragment, _ := strings.Cut(inline.URL, "#")
	page := strings.TrimSuffix(strings.TrimSuffix(path.Base(file), ".markdown"), ".md")
	if file == "" {
		page = ""
	}
	target := page
	if fragment != "" {
		target += "#" + fragment
	}
	if target == "" {
		return label
	}

	return "[[" + target + "|" + label + "]]"
}

func image(inline exportmodel.Inline) string {
	if strings.Contains(inline.URL, "://") {
		return "[" + inline.URL + " " + escape(inline.Text) + "]"
	}

	return "[[File:" + path.Base(inline.URL) + "|" + escape(inline.Text) + "]]"
}

// escape protects what MediaWiki would read as markup: brackets, braces, pipes, doubled quotes
// and the HTML characters.
func escape(text string) string {
	text = escapeHTML(text)
	for _, replacement := range [][2]string{{"[", "&#91;"}, {"]", "&#93;"}, {"{", "&#123;"}, {"}", "&#125;"}, {"|", "&#124;"}, {"''", "&#39;&#39;"}} {
		text = strings.ReplaceAll(text, replacement[0], replacement[1])
	}

	return text
}

func escapeHTML(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}

// startOfLine stops a paragraph that begins with a list, heading or preformatted marker from being
// read as one.
func startOfLine(text string) string {
	if text != "" && strings.ContainsRune("*#:;= ", rune(text[0])) {
		return "&#" + strconv.Itoa(int(text[0])) + ";" + text[1:]
	}

	return text
}
