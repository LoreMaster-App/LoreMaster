package confluenceclient

import "testing"

func TestConfluenceClient(t *testing.T) {
	if got := ConfluenceClient("works"); got != "ConfluenceClient works" {
		t.Fatalf("got %q", got)
	}
}
