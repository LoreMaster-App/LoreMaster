package syncplanning

import (
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/syncannotation"
)

func parseDocument(t *testing.T, content string) documentparsing.MarkdownDocument {
	t.Helper()
	document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath("page.md"), []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	return document
}

// annotated parses body with an annotation recording its current hash, as a sync leaves it.
func syncedDocument(t *testing.T, body string, fields ...string) documentparsing.MarkdownDocument {
	t.Helper()
	hash := syncannotation.ContentHash(parseDocument(t, body).Body)
	block := "<!-- lore-master\npage-id: 42\nversion: 3\ncontent-hash: " + hash + "\n" + strings.Join(fields, "\n") + "\n-->\n"

	return parseDocument(t, block+body)
}

func TestLocalStatusOf(t *testing.T) {
	output := workspacesettings.Output{BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG"}
	const body = "# Page\n\nText.\n"

	edited := syncedDocument(t, body)
	edited.Body = append(edited.Body, []byte("more\n")...)

	cases := []struct {
		name     string
		document documentparsing.MarkdownDocument
		want     LocalStatus
	}{
		{"no annotation", parseDocument(t, body), LocalNew},
		{"annotation without a page id", parseDocument(t, "<!-- lore-master\nversion: 1\n-->\n"+body), LocalNew},
		{"matches the last sync", syncedDocument(t, body), LocalSynced},
		{"same site and space written out", syncedDocument(t, body, "base-url: https://acme.atlassian.net/wiki", "space: ENG"), LocalSynced},
		{"edited since the last sync", edited, LocalChanged},
		{"synced to another space", syncedDocument(t, body, "space: OPS"), LocalNew},
		{"synced to another site", syncedDocument(t, body, "base-url: https://other.atlassian.net/wiki"), LocalNew},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LocalStatusOf(tc.document, output); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
