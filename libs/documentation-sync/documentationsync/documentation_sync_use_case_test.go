package documentationsync

import "testing"

func TestDocumentationSync(t *testing.T) {
	if got := DocumentationSync("works"); got != "DocumentationSync works" {
		t.Fatalf("got %q", got)
	}
}
