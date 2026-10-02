package syncannotation

import (
	"regexp"
	"testing"
)

func TestContentHash(t *testing.T) {
	lf := ContentHash([]byte("# Title\n\nText  \n"))
	if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(lf) {
		t.Fatalf("format %q", lf)
	}
	if got := ContentHash([]byte("# Title\r\n\r\nText  \r\n")); got != lf {
		t.Fatal("CRLF must hash like LF")
	}
	if got := ContentHash([]byte(bom + "# Title\n\nText  \n")); got != lf {
		t.Fatal("a byte-order mark must not change the hash")
	}
	if got := ContentHash([]byte("# Title\n\nText\n")); got == lf {
		t.Fatal("trailing spaces are a Markdown line break and must change the hash")
	}
}
