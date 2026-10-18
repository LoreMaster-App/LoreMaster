// Package latexexport renders a document as a LaTeX file that compiles on its own, for papers and
// anything else a TeX toolchain turns into a PDF.
package latexexport

import (
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

const preamble = `\documentclass{article}
\usepackage[utf8]{inputenc}
\usepackage[T1]{fontenc}
\usepackage{hyperref}
\usepackage{graphicx}
\usepackage{listings}
\usepackage[normalem]{ulem}
\usepackage{amssymb}
\begin{document}
`

// Render writes the document as a complete LaTeX file.
func Render(document exportmodel.Document) string {
	return preamble + "\n" + strings.Join(blocks(document.Blocks), "\n\n") + "\n\n\\end{document}\n"
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

var sections = []string{"section", "subsection", "subsubsection", "paragraph", "subparagraph", "subparagraph"}

func block(current exportmodel.Block) string {
	switch current.Kind {
	case exportmodel.HeadingBlock:
		return "\\" + sections[min(max(current.Level, 1), 6)-1] + "{" + inlines(current.Inlines) + "}"
	case exportmodel.ParagraphBlock:
		return inlines(current.Inlines)
	case exportmodel.ListBlock:
		return list(current)
	case exportmodel.QuoteBlock:
		return "\\begin{quote}\n" + strings.Join(blocks(current.Children), "\n\n") + "\n\\end{quote}"
	case exportmodel.CodeBlock:
		return "\\begin{lstlisting}\n" + current.Text + "\n\\end{lstlisting}"
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return "\\noindent\\rule{\\linewidth}{0.4pt}"
	default:
		return ""
	}
}

func list(current exportmodel.Block) string {
	environment := "itemize"
	if current.Ordered {
		environment = "enumerate"
	}
	lines := []string{"\\begin{" + environment + "}"}
	for _, item := range current.Items {
		box := ""
		if item.Checked != nil {
			box = "$\\square$ "
			if *item.Checked {
				box = "$\\boxtimes$ "
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
		lines = append(lines, "\\item "+strings.Join(parts, "\n"))
	}

	return strings.Join(append(lines, "\\end{"+environment+"}"), "\n")
}

func table(current exportmodel.Block) string {
	columns := len(current.Header)
	for _, row := range current.Rows {
		columns = max(columns, len(row))
	}
	spec := ""
	for column := 0; column < columns; column++ {
		letter := "l"
		if column < len(current.Aligns) {
			switch current.Aligns[column] {
			case exportmodel.AlignCenter:
				letter = "c"
			case exportmodel.AlignRight:
				letter = "r"
			}
		}
		spec += letter
	}
	lines := []string{"\\begin{tabular}{" + spec + "}", "\\hline"}
	row := func(cells []exportmodel.Cell) {
		parts := make([]string, len(cells))
		for index, cell := range cells {
			parts[index] = inlines(cell)
		}
		lines = append(lines, strings.Join(parts, " & ")+" \\\\")
	}
	if len(current.Header) > 0 {
		row(current.Header)
		lines = append(lines, "\\hline")
	}
	for _, cells := range current.Rows {
		row(cells)
	}

	return strings.Join(append(lines, "\\hline", "\\end{tabular}"), "\n")
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString("\\texttt{" + escape(inline.Text) + "}")
		case exportmodel.EmphasisInline:
			out.WriteString("\\emph{" + inlines(inline.Children) + "}")
		case exportmodel.StrongInline:
			out.WriteString("\\textbf{" + inlines(inline.Children) + "}")
		case exportmodel.StrikeInline:
			out.WriteString("\\sout{" + inlines(inline.Children) + "}")
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString("\\includegraphics[width=\\linewidth]{" + inline.URL + "}")
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString("\\\\\n")
		}
	}

	return out.String()
}

// link writes an external address as a hyperlink; a link to another file has no address in a
// compiled document, so only its label stays.
func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if strings.Contains(inline.URL, "://") || strings.HasPrefix(inline.URL, "mailto:") {
		return "\\href{" + strings.NewReplacer("%", "\\%", "#", "\\#").Replace(inline.URL) + "}{" + label + "}"
	}

	return label
}

// escape protects the characters TeX gives a meaning.
func escape(text string) string {
	return strings.NewReplacer(
		"\\", "\\textbackslash{}", "&", "\\&", "%", "\\%", "$", "\\$", "#", "\\#", "_", "\\_",
		"{", "\\{", "}", "\\}", "~", "\\textasciitilde{}", "^", "\\textasciicircum{}",
	).Replace(text)
}
