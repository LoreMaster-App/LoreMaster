// Package slackexport renders a document as Slack mrkdwn, so it can be pasted into a message.
package slackexport

import (
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document as mrkdwn. Slack has no headings, tables or horizontal rules, so a
// heading becomes a bold line, a table a code block and a rule a line of dashes.
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
		return "*" + inlines(current.Inlines) + "*"
	case exportmodel.ParagraphBlock:
		return inlines(current.Inlines)
	case exportmodel.ListBlock:
		return list(current, 0)
	case exportmodel.QuoteBlock:
		return prefix(strings.Join(blocks(current.Children), "\n\n"), "> ")
	case exportmodel.CodeBlock:
		return "```\n" + current.Text + "\n```"
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "-----"
	case exportmodel.HTMLBlock:
		return ""
	default:
		return ""
	}
}

func prefix(text string, lead string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		lines[index] = lead + line
	}

	return strings.Join(lines, "\n")
}

func list(current exportmodel.Block, depth int) string {
	pad := strings.Repeat("    ", depth)
	var lines []string
	for number, item := range current.Items {
		marker := "• "
		if current.Ordered {
			marker = strconv.Itoa(max(current.Start, 1)+number) + ". "
		}
		if item.Checked != nil {
			if *item.Checked {
				marker += "☑ "
			} else {
				marker += "☐ "
			}
		}
		first := true
		for _, inner := range item.Blocks {
			switch inner.Kind {
			case exportmodel.ListBlock:
				lines = append(lines, list(inner, depth+1))
			case exportmodel.ParagraphBlock:
				if first {
					lines = append(lines, pad+marker+inlines(inner.Inlines))
				} else {
					lines = append(lines, pad+"    "+inlines(inner.Inlines))
				}
			default:
				lines = append(lines, pad+"    "+strings.ReplaceAll(block(inner), "\n", " "))
			}
			first = false
		}
	}

	return strings.Join(lines, "\n")
}

// table writes the cells as pipe-separated rows in a code block, which keeps the columns in line.
func table(current exportmodel.Block) string {
	var rows [][]string
	add := func(cells []exportmodel.Cell) {
		row := make([]string, len(cells))
		for index, cell := range cells {
			row[index] = exportmodel.PlainText(cell)
		}
		rows = append(rows, row)
	}
	if len(current.Header) > 0 {
		add(current.Header)
	}
	for _, cells := range current.Rows {
		add(cells)
	}
	widths := map[int]int{}
	for _, row := range rows {
		for index, cell := range row {
			widths[index] = max(widths[index], len([]rune(cell)))
		}
	}
	lines := make([]string, 0, len(rows)+1)
	for index, row := range rows {
		padded := make([]string, len(row))
		for column, cell := range row {
			padded[column] = cell + strings.Repeat(" ", widths[column]-len([]rune(cell)))
		}
		lines = append(lines, strings.TrimRight(strings.Join(padded, " | "), " "))
		if index == 0 && len(current.Header) > 0 {
			rule := make([]string, len(row))
			for column := range row {
				rule[column] = strings.Repeat("-", widths[column])
			}
			lines = append(lines, strings.Join(rule, "-+-"))
		}
	}

	return "```\n" + strings.Join(lines, "\n") + "\n```"
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString("`" + inline.Text + "`")
		case exportmodel.EmphasisInline:
			out.WriteString("_" + inlines(inline.Children) + "_")
		case exportmodel.StrongInline:
			out.WriteString("*" + inlines(inline.Children) + "*")
		case exportmodel.StrikeInline:
			out.WriteString("~" + inlines(inline.Children) + "~")
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString("<" + inline.URL + "|" + escape(inline.Text) + ">")
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("\n")
		}
	}

	return out.String()
}

// link writes an address as <url|label>; a link to a file or heading of the repository has no
// address Slack could open, so only its label stays.
func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if strings.Contains(inline.URL, "://") || strings.HasPrefix(inline.URL, "mailto:") {
		return "<" + inline.URL + "|" + label + ">"
	}

	return label
}

// escape protects the three characters Slack reads as control characters.
func escape(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}
