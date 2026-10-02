package documentparsing

import (
	"testing"
)

func TestHeadingsCarryGitHubSlugs(t *testing.T) {
	document, err := ParseDocument("a.md", []byte("# Lore *Master*\n\n## Getting started!\n\n## Getting started\n\n> ### `go test` & you\n\n## Café 2.0 — notes\n\n## snake_case  twice\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		level      int
		text, slug string
	}{
		{1, "Lore Master", "lore-master"},
		{2, "Getting started!", "getting-started"},
		{2, "Getting started", "getting-started-1"},
		{3, "go test & you", "go-test--you"},
		{2, "Café 2.0 — notes", "café-20--notes"},
		{2, "snake_case twice", "snake_case-twice"},
	}
	got := Headings(document)
	if len(got) != len(want) {
		t.Fatalf("got %d headings: %+v", len(got), got)
	}
	for i, heading := range got {
		if heading.Level != want[i].level || heading.Text != want[i].text || heading.Slug != want[i].slug {
			t.Errorf("heading %d: got %d %q %q, want %+v", i, heading.Level, heading.Text, heading.Slug, want[i])
		}
	}
	if document.TitleHeading != got[0].Node {
		t.Fatal("the title heading is the first H1")
	}

	for fragment, text := range map[string]string{
		"getting-started-1": "Getting started", "Getting-Started": "Getting started!", "caf%C3%A9-20--notes": "Café 2.0 — notes",
	} {
		if heading, ok := FindHeading(got, fragment); !ok || heading.Text != text {
			t.Errorf("FindHeading(%q) = %q %v, want %q", fragment, heading.Text, ok, text)
		}
	}
	if _, ok := FindHeading(got, "nowhere"); ok {
		t.Error("an unknown fragment matches nothing")
	}
}
