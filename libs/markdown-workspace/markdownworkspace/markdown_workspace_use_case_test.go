package markdownworkspace

import "testing"

func TestMarkdownWorkspace(t *testing.T) {
	if got := MarkdownWorkspace("works"); got != "MarkdownWorkspace works" {
		t.Fatalf("got %q", got)
	}
}
