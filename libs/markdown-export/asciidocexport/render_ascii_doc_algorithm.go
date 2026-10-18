// Package asciidocexport renders a document as AsciiDoc, the source format of Asciidoctor and
// Antora sites.
package asciidocexport

import (
	"path"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document as AsciiDoc.
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
		return strings.Repeat("=", min(max(current.Level, 1), 6)) + " " + inlines(current.Inlines)
	case exportmodel.ParagraphBlock:
		return startOfLine(inlines(current.Inlines))
	case exportmodel.ListBlock:
		return list(current, 1)
	case exportmodel.QuoteBlock:
		return "[quote]\n____\n" + strings.Join(blocks(current.Children), "\n\n") + "\n____"
	case exportmodel.CodeBlock:
		return code(current)
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "'''"
	case exportmodel.HTMLBlock:
		return "++++\n" + current.Text + "\n++++"
	default:
		return ""
	}
}

func code(current exportmodel.Block) string {
	header := "[source]"
	if current.Lang != "" {
		header = "[source," + current.Lang + "]"
	}

	return header + "\n----\n" + current.Text + "\n----"
}

// list writes a list whose markers are repeated once per depth, as AsciiDoc nests them.
func list(current exportmodel.Block, depth int) string {
	marker := strings.Repeat("*", depth)
	if current.Ordered {
		marker = strings.Repeat(".", depth)
	}
	var lines []string
	for _, item := range current.Items {
		box := ""
		if item.Checked != nil {
			box = "[ ] "
			if *item.Checked {
				box = "[x] "
			}
		}
		first := true
		for _, inner := range item.Blocks {
			switch inner.Kind {
			case exportmodel.ListBlock:
				lines = append(lines, list(inner, depth+1))
			case exportmodel.ParagraphBlock:
				text := inlines(inner.Inlines)
				if first {
					text = box + text
				}
				lines = append(lines, marker+" "+text)
			default:
				lines = append(lines, marker+" "+strings.ReplaceAll(block(inner), "\n", " "))
			}
			first = false
		}
	}

	return strings.Join(lines, "\n")
}

func table(current exportmodel.Block) string {
	columns := len(current.Header)
	for _, row := range current.Rows {
		columns = max(columns, len(row))
	}
	specs := make([]string, columns)
	for column := range specs {
		specs[column] = "<"
		if column < len(current.Aligns) {
			switch current.Aligns[column] {
			case exportmodel.AlignCenter:
				specs[column] = "^"
			case exportmodel.AlignRight:
				specs[column] = ">"
			}
		}
	}
	options := ""
	if len(current.Header) > 0 {
		options = ",options=\"header\""
	}
	lines := []string{"[cols=\"" + strings.Join(specs, ",") + "\"" + options + "]", "|==="}
	row := func(cells []exportmodel.Cell) {
		for _, cell := range cells {
			lines = append(lines, "|"+strings.ReplaceAll(inlines(cell), "|", "\\|"))
		}
		lines = append(lines, "")
	}
	if len(current.Header) > 0 {
		row(current.Header)
	}
	for _, cells := range current.Rows {
		row(cells)
	}
	lines = append(lines, "|===")

	return strings.Join(lines, "\n")
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString("`+" + inline.Text + "+`")
		case exportmodel.EmphasisInline:
			out.WriteString("__" + inlines(inline.Children) + "__")
		case exportmodel.StrongInline:
			out.WriteString("**" + inlines(inline.Children) + "**")
		case exportmodel.StrikeInline:
			out.WriteString("[.line-through]#" + inlines(inline.Children) + "#")
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString("image:" + inline.URL + "[" + strings.ReplaceAll(inline.Text, "]", "\\]") + "]")
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString(" +\n")
		case exportmodel.HTMLInline:
			out.WriteString("pass:[" + strings.ReplaceAll(inline.Text, "]", "\\]") + "]")
		}
	}

	return out.String()
}

// link writes an external address as url[label] and a link to another Markdown file as a cross
// reference to the AsciiDoc file of the same name.
func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if strings.Contains(inline.URL, "://") || strings.HasPrefix(inline.URL, "mailto:") {
		return inline.URL + "[" + label + "]"
	}
	file, fragment, _ := strings.Cut(inline.URL, "#")
	target := ""
	if file != "" {
		target = strings.TrimSuffix(strings.TrimSuffix(path.Base(file), ".markdown"), ".md") + ".adoc"
	}
	if fragment != "" {
		target += "#" + fragment
	}
	if target == "" {
		return label
	}

	return "xref:" + target + "[" + label + "]"
}

// escape keeps AsciiDoc from reading text as markup: text with a special character is written as
// a passthrough that shows the characters as they are.
func escape(text string) string {
	if !strings.ContainsAny(text, "*_`#+[]{}<>&\\|^~") {
		return text
	}

	return "pass:c[" + strings.ReplaceAll(text, "]", "\\]") + "]"
}

// startOfLine stops a paragraph that begins with a list, heading, block or comment marker from
// being read as one.
func startOfLine(text string) string {
	if text != "" && strings.ContainsRune("=*-.>|+/:[ ", rune(text[0])) {
		return "{empty}" + text
	}

	return text
}
