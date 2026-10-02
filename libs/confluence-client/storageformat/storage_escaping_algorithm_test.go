package storageformat

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestEscapeText(t *testing.T) {
	cases := map[string]string{
		`a & b`:          `a &amp; b`,
		`<tag attr="x">`: `&lt;tag attr=&quot;x&quot;&gt;`,
		`it's`:           `it&#39;s`,
		"tab\tnl\ncr\r":  "tab\tnl\ncr\r",
		"nul\x00vt\x0b":  "nulvt",
		"￾￿":             "",
		"bad\xffutf8":    "badutf8",
		"emoji 🙂 ü":      "emoji 🙂 ü",
	}
	for in, want := range cases {
		if got := escapeText(in); got != want {
			t.Errorf("escapeText(%q) = %q, want %q", in, got, want)
		}
	}
}

// FuzzEscapeText checks the property the escaping exists for: whatever the input,
// the escaped text is well-formed both as character data and as a quoted attribute
// value, and reads back as the input minus the characters XML cannot carry.
// `go test` runs the seeds; `go test -fuzz FuzzEscapeText` explores further.
func FuzzEscapeText(f *testing.F) {
	for _, seed := range []string{"", "plain", `<>&"'`, "]]>", "--", "\x00\x01\x1f", "￾", "\xff\xfe", "🙂", "&amp;", "&#0;"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		escaped := escapeText(input)
		expected := keepXMLCharacters(input)
		for _, document := range []string{"<p>" + escaped + "</p>", `<p a="` + escaped + `"/>`, `<p a='` + escaped + `'/>`} {
			decoder := xml.NewDecoder(strings.NewReader(document))
			var read strings.Builder
			for {
				token, err := decoder.Token()
				if err != nil {
					if err.Error() != "EOF" {
						t.Fatalf("%q → %q is not well-formed: %v", input, document, err)
					}

					break
				}
				switch tok := token.(type) {
				case xml.CharData:
					read.Write(tok)
				case xml.StartElement:
					for _, attribute := range tok.Attr {
						read.WriteString(attribute.Value)
					}
				}
			}
			// The XML parser normalises \r\n and \r to \n in character data, and
			// whitespace to spaces in attribute values; compare modulo that.
			if normalise(read.String()) != normalise(expected) {
				t.Fatalf("%q read back as %q, want %q (document %q)", input, read.String(), expected, document)
			}
		}
	})
}

func normalise(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	return strings.NewReplacer("\n", " ", "\t", " ").Replace(value)
}
