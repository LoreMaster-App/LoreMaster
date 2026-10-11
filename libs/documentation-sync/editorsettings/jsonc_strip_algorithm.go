package editorsettings

// stripJSONC turns the JSON-with-comments an editor's settings.json is into plain JSON:
// line and block comments are dropped and a comma before a closing bracket is removed.
// Strings are left alone, so "//" inside a URL survives. Line breaks stay where they
// were, so a decoder's offsets still point at the right line.
func stripJSONC(source []byte) []byte {
	return dropTrailingCommas(dropComments(source))
}

func dropComments(source []byte) []byte {
	out := make([]byte, 0, len(source))
	inString := false
	for i := 0; i < len(source); i++ {
		c := source[i]
		switch {
		case inString:
			out = append(out, c)
			if c == '\\' && i+1 < len(source) {
				i++
				out = append(out, source[i])
			} else if c == '"' {
				inString = false
			}
		case c == '"':
			inString = true
			out = append(out, c)
		case c == '/' && i+1 < len(source) && source[i+1] == '/':
			for i < len(source) && source[i] != '\n' {
				i++
			}
			if i < len(source) {
				out = append(out, '\n')
			}
		case c == '/' && i+1 < len(source) && source[i+1] == '*':
			i += 2
			for i+1 < len(source) && (source[i] != '*' || source[i+1] != '/') {
				if source[i] == '\n' {
					out = append(out, '\n')
				}
				i++
			}
			i++
		default:
			out = append(out, c)
		}
	}

	return out
}

func dropTrailingCommas(source []byte) []byte {
	out := make([]byte, 0, len(source))
	inString := false
	for i := 0; i < len(source); i++ {
		c := source[i]
		if inString {
			out = append(out, c)
			if c == '\\' && i+1 < len(source) {
				i++
				out = append(out, source[i])
			} else if c == '"' {
				inString = false
			}

			continue
		}
		if c == '"' {
			inString = true
		}
		if c == ',' {
			next := i + 1
			for next < len(source) && (source[next] == ' ' || source[next] == '\t' || source[next] == '\r' || source[next] == '\n') {
				next++
			}
			if next < len(source) && (source[next] == '}' || source[next] == ']') {
				continue
			}
		}
		out = append(out, c)
	}

	return out
}
