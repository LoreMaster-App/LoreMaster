// Package orgexport renders a document as Org-mode, the outline format of Emacs.
package orgexport

import (
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Render writes the document as Org.
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
		return strings.Repeat("*", min(max(current.Level, 1), 6)) + " " + inlines(current.Inlines)
	case exportmodel.ParagraphBlock:
		return startOfLine(inlines(current.Inlines))
	case exportmodel.ListBlock:
		return list(current, 0)
	case exportmodel.QuoteBlock:
		return "#+begin_quote\n" + strings.Join(blocks(current.Children), "\n\n") + "\n#+end_quote"
	case exportmodel.CodeBlock:
		return "#+begin_src " + current.Lang + "\n" + current.Text + "\n#+end_src"
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "-----"
	case exportmodel.HTMLBlock:
		return "#+begin_export html\n" + current.Text + "\n#+end_export"
	default:
		return ""
	}
}

func list(current exportmodel.Block, depth int) string {
	pad := strings.Repeat("  ", depth)
	var lines []string
	for number, item := range current.Items {
		marker := "- "
		if current.Ordered {
			marker = strconv.Itoa(max(current.Start, 1)+number) + ". "
		}
		if item.Checked != nil {
			marker += "[ ] "
			if *item.Checked {
				marker = strings.TrimSuffix(marker, "[ ] ") + "[X] "
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
					lines = append(lines, pad+"  "+inlines(inner.Inlines))
				}
			default:
				lines = append(lines, pad+"  "+strings.ReplaceAll(block(inner), "\n", " "))
			}
			first = false
		}
	}

	return strings.Join(lines, "\n")
}

func table(current exportmodel.Block) string {
	var lines []string
	row := func(cells []exportmodel.Cell) {
		parts := make([]string, len(cells))
		for index, cell := range cells {
			parts[index] = strings.ReplaceAll(inlines(cell), "|", "\\vert{}")
		}
		lines = append(lines, "| "+strings.Join(parts, " | ")+" |")
	}
	if len(current.Header) > 0 {
		row(current.Header)
		lines = append(lines, "|"+strings.Repeat("-+", max(len(current.Header)-1, 0))+"-|")
	}
	for _, cells := range current.Rows {
		row(cells)
	}

	return strings.Join(lines, "\n")
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(inline.Text)
		case exportmodel.CodeInline:
			out.WriteString("~" + inline.Text + "~")
		case exportmodel.EmphasisInline:
			out.WriteString("/" + inlines(inline.Children) + "/")
		case exportmodel.StrongInline:
			out.WriteString("*" + inlines(inline.Children) + "*")
		case exportmodel.StrikeInline:
			out.WriteString("+" + inlines(inline.Children) + "+")
		case exportmodel.LinkInline:
			out.WriteString("[[" + inline.URL + "][" + inlines(inline.Children) + "]]")
		case exportmodel.ImageInline:
			out.WriteString("[[" + inline.URL + "]]")
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("\\\\\n")
		}
	}

	return out.String()
}

// startOfLine stops a paragraph that begins with a heading, list or keyword marker from being
// read as one, by leading it with a zero-width space.
func startOfLine(text string) string {
	if text != "" && strings.ContainsRune("*-+#|:", rune(text[0])) {
		return "​" + text
	}

	return text
}
