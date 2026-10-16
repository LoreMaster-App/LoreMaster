package exportmodel

// PlainText flattens inlines to their text, the way a reader would read them.
func PlainText(inlines []Inline) string {
	text := ""
	for _, inline := range inlines {
		switch inline.Kind {
		case SoftBreakInline, HardBreakInline:
			text += " "
		case TextInline, CodeInline, HTMLInline, ImageInline:
			if inline.Kind == HTMLInline {
				continue
			}
			text += inline.Text
		default:
			text += PlainText(inline.Children)
		}
	}

	return text
}
