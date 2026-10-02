package storageformat

import (
	"strings"
	"unicode/utf8"
)

// escapeText makes text safe as XHTML character data and as a double- or
// single-quoted attribute value: & < > " ' become references, and characters XML 1.0
// cannot carry at all (C0 controls other than tab, newline and carriage return, lone
// surrogates, U+FFFE/U+FFFF, invalid UTF-8) are dropped, since Confluence rejects a
// document containing any of them.
func escapeText(value string) string {
	var out strings.Builder
	out.Grow(len(value))
	for i := 0; i < len(value); {
		r, size := utf8.DecodeRuneInString(value[i:])
		i += size
		switch {
		case r == utf8.RuneError && size == 1:
			continue
		case !allowedInXML(r):
			continue
		case r == '&':
			out.WriteString("&amp;")
		case r == '<':
			out.WriteString("&lt;")
		case r == '>':
			out.WriteString("&gt;")
		case r == '"':
			out.WriteString("&quot;")
		case r == '\'':
			out.WriteString("&#39;")
		default:
			out.WriteRune(r)
		}
	}

	return out.String()
}

// allowedInXML is XML 1.0's Char production.
func allowedInXML(r rune) bool {
	return r == '\t' || r == '\n' || r == '\r' ||
		(r >= 0x20 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0x10FFFF)
}

// keepXMLCharacters drops what XML 1.0 cannot carry and leaves the rest untouched, for
// CDATA, where nothing else needs escaping.
func keepXMLCharacters(value string) string {
	var out strings.Builder
	out.Grow(len(value))
	for i := 0; i < len(value); {
		r, size := utf8.DecodeRuneInString(value[i:])
		i += size
		if (r != utf8.RuneError || size != 1) && allowedInXML(r) {
			out.WriteRune(r)
		}
	}

	return out.String()
}
