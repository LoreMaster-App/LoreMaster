// Package jirawikiexport renders a document as Jira's wiki markup, the text format of Jira Server,
// Data Center and the legacy editor of Jira Cloud.
package jirawikiexport

import (
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document as Jira wiki markup.
func Render(document exportmodel.Document) string {
	return strings.Join(blocks(document.Blocks, ""), "\n\n") + "\n"
}

func blocks(list []exportmodel.Block, listPrefix string) []string {
	rendered := make([]string, 0, len(list))
	for _, current := range list {
		if text := block(current, listPrefix); text != "" {
			rendered = append(rendered, text)
		}
	}

	return rendered
}

func block(current exportmodel.Block, listPrefix string) string {
	switch current.Kind {
	case exportmodel.HeadingBlock:
		return "h" + strconv.Itoa(min(max(current.Level, 1), 6)) + ". " + inlines(current.Inlines)
	case exportmodel.ParagraphBlock:
		return inlines(current.Inlines)
	case exportmodel.ListBlock:
		return list(current, listPrefix)
	case exportmodel.QuoteBlock:
		return "{quote}\n" + strings.Join(blocks(current.Children, ""), "\n\n") + "\n{quote}"
	case exportmodel.CodeBlock:
		opening := "{code}"
		if current.Lang != "" {
			opening = "{code:" + current.Lang + "}"
		}

		return opening + "\n" + current.Text + "\n{code}"
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "----"
	case exportmodel.HTMLBlock:
		return "{noformat}\n" + current.Text + "\n{noformat}"
	default:
		return ""
	}
}

// list writes one item per line; nested lists repeat the marker for each level they sit at.
func list(current exportmodel.Block, parent string) string {
	marker := "*"
	if current.Ordered {
		marker = "#"
	}
	prefix := parent + marker
	var lines []string
	for _, item := range current.Items {
		icon := ""
		if item.Checked != nil {
			icon = "(x) "
			if *item.Checked {
				icon = "(/) "
			}
		}
		first := true
		for _, inner := range item.Blocks {
			switch inner.Kind {
			case exportmodel.ListBlock:
				lines = append(lines, list(inner, prefix))
			case exportmodel.ParagraphBlock:
				if first {
					lines = append(lines, prefix+" "+icon+inlines(inner.Inlines))
				} else {
					lines = append(lines, prefix+" "+inlines(inner.Inlines))
				}
			default:
				lines = append(lines, prefix+" "+strings.ReplaceAll(block(inner, ""), "\n", " "))
			}
			first = false
		}
	}

	return strings.Join(lines, "\n")
}

func table(current exportmodel.Block) string {
	var lines []string
	if len(current.Header) > 0 {
		lines = append(lines, "||"+strings.Join(cells(current.Header), "||")+"||")
	}
	for _, row := range current.Rows {
		lines = append(lines, "|"+strings.Join(cells(row), "|")+"|")
	}

	return strings.Join(lines, "\n")
}

func cells(row []exportmodel.Cell) []string {
	texts := make([]string, len(row))
	for i, cell := range row {
		text := inlines(cell)
		if strings.TrimSpace(text) == "" {
			text = " "
		}
		texts[i] = text
	}

	return texts
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString("{{" + strings.ReplaceAll(inline.Text, "}}", "} }") + "}}")
		case exportmodel.EmphasisInline:
			out.WriteString("_" + inlines(inline.Children) + "_")
		case exportmodel.StrongInline:
			out.WriteString("*" + inlines(inline.Children) + "*")
		case exportmodel.StrikeInline:
			out.WriteString("-" + inlines(inline.Children) + "-")
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString("!" + inline.URL + "!")
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("\\\\ ")
		case exportmodel.HTMLInline:
		}
	}

	return out.String()
}

func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if inline.URL == "" {
		return label
	}
	if label == "" || label == escape(inline.URL) {
		return "[" + inline.URL + "]"
	}

	return "[" + label + "|" + inline.URL + "]"
}

// escape protects the characters Jira would read as markup.
func escape(text string) string {
	var out strings.Builder
	for _, character := range text {
		if strings.ContainsRune(`\{}[]|*_~^`, character) {
			out.WriteByte('\\')
		}
		out.WriteRune(character)
	}

	return out.String()
}
