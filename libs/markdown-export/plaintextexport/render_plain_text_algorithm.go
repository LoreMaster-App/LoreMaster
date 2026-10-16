// Package plaintextexport renders a document as plain text: the Markdown without its syntax.
package plaintextexport

import (
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document's text. Headings stand on their own line (the first two levels
// underlined), lists keep their bullets and numbers, a table is laid out in aligned columns,
// a link keeps its address in brackets, an image is its alt text and code is kept as written.
func Render(document exportmodel.Document) string {
	return strings.Join(renderBlocks(document.Blocks), "\n\n") + "\n"
}

func renderBlocks(blocks []exportmodel.Block) []string {
	rendered := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if text := renderBlock(block); text != "" {
			rendered = append(rendered, text)
		}
	}

	return rendered
}

func renderBlock(block exportmodel.Block) string {
	switch block.Kind {
	case exportmodel.HeadingBlock:
		return heading(block)
	case exportmodel.ParagraphBlock:
		return inlineText(block.Inlines)
	case exportmodel.ListBlock:
		return list(block)
	case exportmodel.QuoteBlock:
		return indent(strings.Join(renderBlocks(block.Children), "\n\n"), "  ")
	case exportmodel.CodeBlock:
		return block.Text
	case exportmodel.TableBlock:
		return table(block)
	case exportmodel.RuleBlock:
		return strings.Repeat("-", 40)
	default:
		return ""
	}
}

func heading(block exportmodel.Block) string {
	text := inlineText(block.Inlines)
	switch block.Level {
	case 1:
		return text + "\n" + strings.Repeat("=", len([]rune(text)))
	case 2:
		return text + "\n" + strings.Repeat("-", len([]rune(text)))
	default:
		return text
	}
}

func list(block exportmodel.Block) string {
	lines := make([]string, 0, len(block.Items))
	number := block.Start
	if number == 0 {
		number = 1
	}
	for _, item := range block.Items {
		marker := "- "
		if block.Ordered {
			marker = strconv.Itoa(number) + ". "
			number++
		}
		width := len(marker)
		if item.Checked != nil {
			marker += "[ ] "
			if *item.Checked {
				marker = marker[:width] + "[x] "
			}
		}
		body := strings.Join(renderBlocks(item.Blocks), "\n")
		lines = append(lines, marker+indentRest(body, strings.Repeat(" ", width)))
	}

	return strings.Join(lines, "\n")
}

func table(block exportmodel.Block) string {
	rows := make([][]string, 0, len(block.Rows)+1)
	if len(block.Header) > 0 {
		rows = append(rows, cellTexts(block.Header))
	}
	for _, row := range block.Rows {
		rows = append(rows, cellTexts(row))
	}
	widths := columnWidths(rows)
	lines := make([]string, 0, len(rows)+1)
	for i, row := range rows {
		cells := make([]string, len(widths))
		for column := range widths {
			value := ""
			if column < len(row) {
				value = row[column]
			}
			cells[column] = value + strings.Repeat(" ", widths[column]-len([]rune(value)))
		}
		lines = append(lines, strings.TrimRight(strings.Join(cells, "  "), " "))
		if i == 0 && len(block.Header) > 0 {
			rules := make([]string, len(widths))
			for column, width := range widths {
				rules[column] = strings.Repeat("-", width)
			}
			lines = append(lines, strings.Join(rules, "  "))
		}
	}

	return strings.Join(lines, "\n")
}

func cellTexts(cells []exportmodel.Cell) []string {
	texts := make([]string, len(cells))
	for i, cell := range cells {
		texts[i] = inlineText(cell)
	}

	return texts
}

func columnWidths(rows [][]string) []int {
	var widths []int
	for _, row := range rows {
		for column, value := range row {
			if column == len(widths) {
				widths = append(widths, 0)
			}
			widths[column] = max(widths[column], len([]rune(value)))
		}
	}

	return widths
}

func inlineText(inlines []exportmodel.Inline) string {
	var text strings.Builder
	for _, inline := range inlines {
		switch inline.Kind {
		case exportmodel.TextInline, exportmodel.CodeInline, exportmodel.ImageInline:
			text.WriteString(inline.Text)
		case exportmodel.SoftBreakInline:
			text.WriteString(" ")
		case exportmodel.HardBreakInline:
			text.WriteString("\n")
		case exportmodel.LinkInline:
			label := inlineText(inline.Children)
			text.WriteString(label)
			if inline.URL != "" && inline.URL != label && !strings.HasPrefix(inline.URL, "#") {
				text.WriteString(" (" + inline.URL + ")")
			}
		case exportmodel.HTMLInline:
		default:
			text.WriteString(inlineText(inline.Children))
		}
	}

	return text.String()
}

func indent(text, prefix string) string {
	return prefix + indentRest(text, prefix)
}

// indentRest prefixes every line but the first.
func indentRest(text, prefix string) string {
	return strings.ReplaceAll(text, "\n", "\n"+prefix)
}
