// Package restexport renders a document as reStructuredText, the source format of Sphinx
// projects and Python documentation.
package restexport

import (
	"path"
	"strings"
	"unicode/utf8"

	"lore-master/libs/markdown-export/exportmodel"
)

// underlines are the heading adornments, one per level, so a document is consistent: every level
// always gets the same one.
var underlines = []string{"=", "-", "~", "^", "\"", "'"}

// Render writes the document as reStructuredText.
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
		text := inlines(current.Inlines)

		return text + "\n" + strings.Repeat(underlines[min(max(current.Level, 1), 6)-1], max(utf8.RuneCountInString(text), 3))
	case exportmodel.ParagraphBlock:
		return paragraph(current)
	case exportmodel.ListBlock:
		return list(current)
	case exportmodel.QuoteBlock:
		return indent(strings.Join(blocks(current.Children), "\n\n"), "    ")
	case exportmodel.CodeBlock:
		language := current.Lang
		if language == "" {
			language = "text"
		}

		return ".. code-block:: " + language + "\n\n" + indent(current.Text, "   ")
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "----"
	case exportmodel.HTMLBlock:
		return ".. raw:: html\n\n" + indent(current.Text, "   ")
	default:
		return ""
	}
}

// paragraph writes a paragraph; one that is only an image is written as an image directive.
func paragraph(current exportmodel.Block) string {
	if len(current.Inlines) == 1 && current.Inlines[0].Kind == exportmodel.ImageInline {
		image := current.Inlines[0]
		text := ".. image:: " + image.URL
		if image.Text != "" {
			text += "\n   :alt: " + image.Text
		}

		return text
	}

	return inlines(current.Inlines)
}

func list(current exportmodel.Block) string {
	marker := "- "
	if current.Ordered {
		marker = "#. "
	}
	pad := strings.Repeat(" ", len(marker))
	var lines []string
	for _, item := range current.Items {
		box := ""
		if item.Checked != nil {
			box = "[ ] "
			if *item.Checked {
				box = "[x] "
			}
		}
		var parts []string
		for index, inner := range item.Blocks {
			text := block(inner)
			if index == 0 && inner.Kind == exportmodel.ParagraphBlock {
				text = box + text
			}
			parts = append(parts, text)
		}
		lines = append(lines, marker+strings.TrimPrefix(indent(strings.Join(parts, "\n\n"), pad), pad))
	}

	return strings.Join(lines, "\n")
}

// table writes a list-table, which has no column widths to keep in step.
func table(current exportmodel.Block) string {
	lines := []string{".. list-table::"}
	if len(current.Header) > 0 {
		lines = append(lines, "   :header-rows: 1")
	}
	lines = append(lines, "")
	row := func(cells []exportmodel.Cell) {
		for column, cell := range cells {
			lead := "     - "
			if column == 0 {
				lead = "   * - "
			}
			lines = append(lines, lead+strings.ReplaceAll(inlines(cell), "\n", " "))
		}
	}
	if len(current.Header) > 0 {
		row(current.Header)
	}
	for _, cells := range current.Rows {
		row(cells)
	}

	return strings.Join(lines, "\n")
}

func indent(text string, pad string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		if line != "" {
			lines[index] = pad + line
		}
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
			out.WriteString("``" + inline.Text + "``")
		case exportmodel.EmphasisInline:
			out.WriteString("*" + inlines(inline.Children) + "*")
		case exportmodel.StrongInline:
			out.WriteString("**" + inlines(inline.Children) + "**")
		case exportmodel.StrikeInline:
			// reStructuredText has no strike-through, so the text stays as it is.
			out.WriteString(inlines(inline.Children))
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString("`" + escape(inline.Text) + " <" + inline.URL + ">`__")
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("\n")
		case exportmodel.HTMLInline:
			out.WriteString(escape(inline.Text))
		}
	}

	return out.String()
}

// link writes an external address as an anonymous hyperlink and a link to another Markdown file
// as a Sphinx document reference.
func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if strings.Contains(inline.URL, "://") || strings.HasPrefix(inline.URL, "mailto:") {
		return "`" + label + " <" + inline.URL + ">`__"
	}
	file, _, _ := strings.Cut(inline.URL, "#")
	if file == "" {
		return label
	}
	page := strings.TrimSuffix(strings.TrimSuffix(path.Base(file), ".markdown"), ".md")

	return ":doc:`" + label + " <" + page + ">`"
}

// escape keeps reStructuredText from reading text as markup.
func escape(text string) string {
	return strings.NewReplacer("\\", "\\\\", "*", "\\*", "`", "\\`", "|", "\\|", "_", "\\_").Replace(text)
}
