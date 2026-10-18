// Package rtfexport renders a document as Rich Text Format, which Word, WordPad and most other
// word processors open.
package rtfexport

import (
	"strconv"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

const header = `{\rtf1\ansi\ansicpg1252\deff0{\fonttbl{\f0 Calibri;}{\f1 Consolas;}}\fs22` + "\n"

// headingSizes are the half-point sizes of heading levels 1 to 6.
var headingSizes = []int{40, 34, 30, 26, 24, 22}

// Render writes the document as RTF.
func Render(document exportmodel.Document) string {
	return header + strings.Join(blocks(document.Blocks, 0), "\n") + "\n}\n"
}

func blocks(list []exportmodel.Block, indent int) []string {
	var rendered []string
	for _, current := range list {
		if text := block(current, indent); text != "" {
			rendered = append(rendered, text)
		}
	}

	return rendered
}

// twips is the indent of a nesting level, in twentieths of a point.
func twips(level int) string {
	return strconv.Itoa(level * 360)
}

func block(current exportmodel.Block, indent int) string {
	switch current.Kind {
	case exportmodel.HeadingBlock:
		size := headingSizes[min(max(current.Level, 1), 6)-1]

		return `\pard\sb240\sa120\li` + twips(indent) + `\b\fs` + strconv.Itoa(size) + " " + inlines(current.Inlines) + `\b0\fs22\par`
	case exportmodel.ParagraphBlock:
		return `\pard\sa120\li` + twips(indent) + " " + inlines(current.Inlines) + `\par`
	case exportmodel.ListBlock:
		return list(current, indent)
	case exportmodel.QuoteBlock:
		return strings.Join(blocks(current.Children, indent+2), "\n")
	case exportmodel.CodeBlock:
		return `\pard\sa120\li` + twips(indent+1) + `{\f1\fs20 ` + strings.ReplaceAll(escape(current.Text), "\n", `\line `) + `}\par`
	case exportmodel.TableBlock:
		return table(current)
	case exportmodel.RuleBlock:
		return `\pard\brdrb\brdrs\brdrw10\brsp20\par`
	default:
		return ""
	}
}

func list(current exportmodel.Block, indent int) string {
	var lines []string
	for number, item := range current.Items {
		marker := `\bullet`
		if current.Ordered {
			marker = strconv.Itoa(max(current.Start, 1)+number) + "."
		}
		if item.Checked != nil {
			marker = `靄?`
			if *item.Checked {
				marker = `靅?`
			}
		}
		first := true
		for _, inner := range item.Blocks {
			switch inner.Kind {
			case exportmodel.ListBlock:
				lines = append(lines, list(inner, indent+1))
			case exportmodel.ParagraphBlock:
				if first {
					lines = append(lines, `\pard\li`+twips(indent+1)+`\fi-360 `+marker+`\tab `+inlines(inner.Inlines)+`\par`)
				} else {
					lines = append(lines, `\pard\li`+twips(indent+1)+" "+inlines(inner.Inlines)+`\par`)
				}
			default:
				lines = append(lines, block(inner, indent+1))
			}
			first = false
		}
	}

	return strings.Join(lines, "\n")
}

// table writes each row with its cell borders; the header row is bold.
func table(current exportmodel.Block) string {
	columns := len(current.Header)
	for _, row := range current.Rows {
		columns = max(columns, len(row))
	}
	if columns == 0 {
		return ""
	}
	width := 9000 / columns
	var edges strings.Builder
	for column := 1; column <= columns; column++ {
		edges.WriteString(`\clbrdrt\brdrs\clbrdrl\brdrs\clbrdrb\brdrs\clbrdrr\brdrs\cellx` + strconv.Itoa(column*width))
	}
	var lines []string
	row := func(cells []exportmodel.Cell, bold bool) {
		line := `\trowd` + edges.String()
		for _, cell := range cells {
			text := inlines(cell)
			if bold {
				text = `{\b ` + text + `}`
			}
			line += `\pard\intbl ` + text + `\cell`
		}
		lines = append(lines, line+`\row`)
	}
	if len(current.Header) > 0 {
		row(current.Header, true)
	}
	for _, cells := range current.Rows {
		row(cells, false)
	}

	return strings.Join(lines, "\n") + "\n" + `\pard\par`
}

func inlines(list []exportmodel.Inline) string {
	var out strings.Builder
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.CodeInline:
			out.WriteString(`{\f1 ` + escape(inline.Text) + `}`)
		case exportmodel.EmphasisInline:
			out.WriteString(`{\i ` + inlines(inline.Children) + `}`)
		case exportmodel.StrongInline:
			out.WriteString(`{\b ` + inlines(inline.Children) + `}`)
		case exportmodel.StrikeInline:
			out.WriteString(`{\strike ` + inlines(inline.Children) + `}`)
		case exportmodel.LinkInline:
			out.WriteString(link(inline))
		case exportmodel.ImageInline:
			out.WriteString(escape(inline.Text))
		case exportmodel.SoftBreakInline:
			out.WriteString(" ")
		case exportmodel.HardBreakInline:
			out.WriteString(`\line `)
		}
	}

	return out.String()
}

// link writes an external address as a clickable field; any other link keeps only its label.
func link(inline exportmodel.Inline) string {
	label := inlines(inline.Children)
	if strings.Contains(inline.URL, "://") || strings.HasPrefix(inline.URL, "mailto:") {
		return `{\field{\*\fldinst{HYPERLINK "` + strings.ReplaceAll(inline.URL, `"`, "%22") + `"}}{\fldrslt{\ul ` + label + `}}}`
	}

	return label
}

// escape protects the three control characters of RTF and writes anything outside ASCII as a
// signed 16-bit code, followed by the question mark a reader that lacks Unicode shows.
func escape(text string) string {
	var out strings.Builder
	for _, character := range text {
		switch {
		case character == '\\' || character == '{' || character == '}':
			out.WriteByte('\\')
			out.WriteRune(character)
		case character == '\n':
			out.WriteString(`\line `)
		case character < 128:
			out.WriteRune(character)
		case character < 0x10000:
			code := int(character)
			if code > 32767 {
				code -= 65536
			}
			out.WriteString(`\u` + strconv.Itoa(code) + "?")
		default:
			high, low := surrogates(character)
			out.WriteString(`\u` + strconv.Itoa(high) + `?\u` + strconv.Itoa(low) + "?")
		}
	}

	return out.String()
}

// surrogates splits a character beyond the basic plane into the pair of signed 16-bit codes RTF
// expects.
func surrogates(character rune) (int, int) {
	offset := int(character) - 0x10000
	high := 0xD800 + offset>>10
	low := 0xDC00 + offset&0x3FF

	return high - 65536, low - 65536
}
