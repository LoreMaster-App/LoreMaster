// Package manexport renders a document as a roff man page (man(7) macros), the format of
// Unix manual pages.
package manexport

import (
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document as a man page in section 7. The first level-1 heading is the page's
// title; every other heading is a section or a subsection.
func Render(document exportmodel.Document) string {
	title := strings.ToUpper(strings.NewReplacer(" ", "-", "\"", "").Replace(exportmodel.PlainText(titleOf(document))))
	lines := []string{".TH \"" + title + "\" \"7\""}
	lines = append(lines, blocks(skipTitle(document.Blocks))...)

	return strings.Join(lines, "\n") + "\n"
}

func titleOf(document exportmodel.Document) []exportmodel.Inline {
	for _, current := range document.Blocks {
		if current.Kind == exportmodel.HeadingBlock && current.Level == 1 {
			return current.Inlines
		}
	}

	return []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: document.Title}}
}

// skipTitle drops the first level-1 heading, which the .TH line already carries.
func skipTitle(list []exportmodel.Block) []exportmodel.Block {
	for index, current := range list {
		if current.Kind == exportmodel.HeadingBlock && current.Level == 1 {
			return append(append([]exportmodel.Block(nil), list[:index]...), list[index+1:]...)
		}
	}

	return list
}

func blocks(list []exportmodel.Block) []string {
	var rendered []string
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
		macro := ".SH"
		if current.Level > 2 {
			macro = ".SS"
		}
		text := inlines(current.Inlines)
		if macro == ".SH" {
			text = strings.ToUpper(text)
		}

		return macro + " \"" + strings.ReplaceAll(text, "\"", "") + "\""
	case exportmodel.ParagraphBlock:
		return ".PP\n" + startOfLine(inlines(current.Inlines))
	case exportmodel.ListBlock:
		return list(current)
	case exportmodel.QuoteBlock:
		return ".RS\n" + strings.Join(blocks(current.Children), "\n") + "\n.RE"
	case exportmodel.CodeBlock:
		return ".PP\n.nf\n.RS\n" + verbatim(current.Text) + "\n.RE\n.fi"
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return ".PP\n\\(em\\(em\\(em"
	default:
		return ""
	}
}

func list(current exportmodel.Block) string {
	var lines []string
	for number, item := range current.Items {
		label := "\\(bu"
		width := "2"
		if current.Ordered {
			label = strconv.Itoa(max(current.Start, 1)+number) + "."
			width = "4"
		}
		if item.Checked != nil {
			label = "[ ]"
			if *item.Checked {
				label = "[x]"
			}
			width = "4"
		}
		lines = append(lines, ".IP "+label+" "+width)
		for _, inner := range item.Blocks {
			switch inner.Kind {
			case exportmodel.ParagraphBlock:
				lines = append(lines, startOfLine(inlines(inner.Inlines)))
			case exportmodel.ListBlock:
				lines = append(lines, ".RS\n"+list(inner)+"\n.RE")
			default:
				lines = append(lines, block(inner))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// table writes a tbl(1) table; a header row is separated from the rows by a rule.
func table(current exportmodel.Block) string {
	columns := len(current.Header)
	for _, row := range current.Rows {
		columns = max(columns, len(row))
	}
	spec := make([]string, columns)
	for column := range spec {
		spec[column] = "l"
		if column < len(current.Aligns) {
			switch current.Aligns[column] {
			case exportmodel.AlignCenter:
				spec[column] = "c"
			case exportmodel.AlignRight:
				spec[column] = "r"
			}
		}
	}
	lines := []string{".TS", "allbox;", strings.Join(spec, " ") + "."}
	row := func(cells []exportmodel.Cell) {
		parts := make([]string, len(cells))
		for index, cell := range cells {
			parts[index] = strings.ReplaceAll(inlines(cell), "\t", " ")
		}
		lines = append(lines, strings.Join(parts, "\t"))
	}
	if len(current.Header) > 0 {
		row(current.Header)
		lines = append(lines, "_")
	}
	for _, cells := range current.Rows {
		row(cells)
	}

	return strings.Join(append(lines, ".TE"), "\n")
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString("\\fB" + escape(inline.Text) + "\\fR")
		case exportmodel.EmphasisInline:
			out.WriteString("\\fI" + inlines(inline.Children) + "\\fR")
		case exportmodel.StrongInline:
			out.WriteString("\\fB" + inlines(inline.Children) + "\\fR")
		case exportmodel.StrikeInline:
			out.WriteString(inlines(inline.Children))
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("\n.br\n")
		}
	}

	return out.String()
}

// link writes the label, and the address after it when there is one a reader could use.
func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if strings.Contains(inline.URL, "://") || strings.HasPrefix(inline.URL, "mailto:") {
		return label + " (\\fI" + escape(inline.URL) + "\\fR)"
	}

	return label
}

// verbatim escapes each line of a code block.
func verbatim(text string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		lines[index] = startOfLine(escape(line))
	}

	return strings.Join(lines, "\n")
}

// escape protects the backslash and the hyphen, which roff reads as an escape and a minus sign.
func escape(text string) string {
	return strings.NewReplacer("\\", "\\e", "-", "\\-").Replace(text)
}

// startOfLine stops a line that begins with a dot or an apostrophe from being read as a macro.
func startOfLine(text string) string {
	if text != "" && (text[0] == '.' || text[0] == '\'') {
		return "\\&" + text
	}

	return text
}
