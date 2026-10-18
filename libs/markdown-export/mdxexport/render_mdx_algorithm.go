// Package mdxexport renders a document as MDX for Docusaurus and similar sites: Markdown with a
// front matter block, and the characters MDX would read as JSX escaped.
package mdxexport

import (
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document as MDX. Raw HTML is left out, because in MDX it is JSX and a stray
// tag breaks the build.
func Render(document exportmodel.Document) string {
	front := "---\ntitle: " + quote(document.Title) + "\n---\n\n"

	return front + strings.Join(blocks(document.Blocks), "\n\n") + "\n"
}

func quote(text string) string {
	return strconv.Quote(text)
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
		return strings.Repeat("#", min(max(current.Level, 1), 6)) + " " + inlines(current.Inlines)
	case exportmodel.ParagraphBlock:
		return inlines(current.Inlines)
	case exportmodel.ListBlock:
		return list(current, "")
	case exportmodel.QuoteBlock:
		return prefix(strings.Join(blocks(current.Children), "\n\n"), "> ")
	case exportmodel.CodeBlock:
		return fence(current.Text) + current.Lang + "\n" + current.Text + "\n" + fence(current.Text)
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "---"
	default:
		return ""
	}
}

// fence is a run of backticks longer than any inside the code.
func fence(text string) string {
	longest, run := 0, 0
	for _, character := range text {
		if character == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}

	return strings.Repeat("`", max(3, longest+1))
}

func prefix(text string, lead string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(lead+line, " ")
	}

	return strings.Join(lines, "\n")
}

func list(current exportmodel.Block, pad string) string {
	var lines []string
	for number, item := range current.Items {
		marker := "- "
		if current.Ordered {
			marker = strconv.Itoa(max(current.Start, 1)+number) + ". "
		}
		if item.Checked != nil {
			marker += "[ ] "
			if *item.Checked {
				marker = strings.TrimSuffix(marker, "[ ] ") + "[x] "
			}
		}
		inner := pad + strings.Repeat(" ", len(marker))
		first := true
		for _, content := range item.Blocks {
			switch content.Kind {
			case exportmodel.ListBlock:
				lines = append(lines, list(content, inner))
			case exportmodel.ParagraphBlock:
				if first {
					lines = append(lines, pad+marker+inlines(content.Inlines))
				} else {
					lines = append(lines, inner+inlines(content.Inlines))
				}
			default:
				lines = append(lines, prefix(block(content), inner))
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
	line := func(cells []exportmodel.Cell) string {
		parts := make([]string, columns)
		for index := range parts {
			if index < len(cells) {
				parts[index] = strings.ReplaceAll(inlines(cells[index]), "|", "\\|")
			}
		}

		return "| " + strings.Join(parts, " | ") + " |"
	}
	rule := make([]string, columns)
	for index := range rule {
		rule[index] = "---"
		if index < len(current.Aligns) {
			switch current.Aligns[index] {
			case exportmodel.AlignLeft:
				rule[index] = ":--"
			case exportmodel.AlignCenter:
				rule[index] = ":-:"
			case exportmodel.AlignRight:
				rule[index] = "--:"
			}
		}
	}
	lines := []string{line(current.Header), "| " + strings.Join(rule, " | ") + " |"}
	for _, cells := range current.Rows {
		lines = append(lines, line(cells))
	}

	return strings.Join(lines, "\n")
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
			out.WriteString("*" + inlines(inline.Children) + "*")
		case exportmodel.StrongInline:
			out.WriteString("**" + inlines(inline.Children) + "**")
		case exportmodel.StrikeInline:
			out.WriteString("~~" + inlines(inline.Children) + "~~")
		case exportmodel.LinkInline:
			out.WriteString("[" + inlines(inline.Children) + "](" + inline.URL + ")")
		case exportmodel.ImageInline:
			out.WriteString("![" + escape(inline.Text) + "](" + inline.URL + ")")
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("<br />\n")
		}
	}

	return out.String()
}

// escape protects what MDX reads as JSX or an expression, and what Markdown reads as emphasis.
func escape(text string) string {
	return strings.NewReplacer("\\", "\\\\", "{", "\\{", "}", "\\}", "<", "&lt;", ">", "&gt;", "*", "\\*", "_", "\\_", "`", "\\`").Replace(text)
}
