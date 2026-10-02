package syncannotation

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fullAnnotation() Annotation {
	return Annotation{
		Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG",
		PageID: "123456", ParentID: "123000", Version: 7, ContentHash: "sha256:abc",
		Attachments: map[string]string{"b.svg": "sha256:2", "a.svg": "sha256:1"},
		SyncedAt:    time.Date(2026, 10, 1, 14, 0, 0, 0, time.FixedZone("CEST", 2*60*60)),
		Title:       "Override",
	}
}

const fullBlockLF = "<!-- lore-master\n" +
	"platform: confluence\n" +
	"base-url: https://acme.atlassian.net/wiki\n" +
	"space: ENG\n" +
	"page-id: 123456\n" +
	"parent-id: 123000\n" +
	"version: 7\n" +
	"content-hash: sha256:abc\n" +
	"attachments: {\"a.svg\":\"sha256:1\",\"b.svg\":\"sha256:2\"}\n" +
	"synced-at: 2026-10-01T12:00:00Z\n" +
	"title: Override\n" +
	"-->\n"

func TestRenderRoundTrip(t *testing.T) {
	fullBlockCRLF := strings.ReplaceAll(fullBlockLF, "\n", "\r\n")
	cases := []struct{ name, original, want string }{
		{"LF", "# Title\n\nText\n", fullBlockLF + "# Title\n\nText\n"},
		{"CRLF", "# Title\r\n\r\nText\r\n", fullBlockCRLF + "# Title\r\n\r\nText\r\n"},
		{"BOM and CRLF", bom + "# Title\r\nText", bom + fullBlockCRLF + "# Title\r\nText"},
		{"empty file", "", fullBlockLF},
		{"front matter", "---\ntags: [a]\n---\n# Title\n", "---\ntags: [a]\n---\n" + fullBlockLF + "# Title\n"},
		{"front matter, nothing after it", "---\na: b\n---", "---\na: b\n---\n" + strings.TrimSuffix(fullBlockLF, "\n")},
		{"replaces an existing block", "<!-- lore-master\npage-id: 1\nversion: 1\n-->\n# Title\n", fullBlockLF + "# Title\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered, err := Render([]byte(tc.original), fullAnnotation())
			if err != nil {
				t.Fatal(err)
			}
			if string(rendered) != tc.want {
				t.Fatalf("rendered\n got: %q\nwant: %q", rendered, tc.want)
			}

			before, err := Read([]byte(tc.original))
			if err != nil {
				t.Fatal(err)
			}
			after, err := Read(rendered)
			if err != nil {
				t.Fatal(err)
			}
			if string(after.Body) != string(before.Body) || after.Layout != before.Layout {
				t.Fatalf("body or layout changed: %q %+v -> %q %+v", before.Body, before.Layout, after.Body, after.Layout)
			}
			if ContentHash(after.Body) != ContentHash(before.Body) {
				t.Fatal("writing the annotation changed the content hash")
			}
			if !reflect.DeepEqual(*after.Annotation, normalised(fullAnnotation())) {
				t.Fatalf("annotation read back as %+v", *after.Annotation)
			}

			again, err := Render(rendered, fullAnnotation())
			if err != nil {
				t.Fatal(err)
			}
			if string(again) != string(rendered) {
				t.Fatalf("rendering the same annotation twice is not a no-op: %q", again)
			}
		})
	}
}

func TestRenderLeavesAnEquivalentBlockAlone(t *testing.T) {
	content := "<!--   lore-master\n" // not the opening line: no annotation at all
	if _, err := Render([]byte(content), Annotation{PageID: "1"}); err != nil {
		t.Fatal(err)
	}
	authored := "<!-- lore-master  \ntitle:   Override \npage-id:123456\n\n-->\n# Title\n"
	rendered, err := Render([]byte(authored), Annotation{PageID: "123456", Title: "Override"})
	if err != nil {
		t.Fatal(err)
	}
	if string(rendered) != authored {
		t.Fatalf("an equivalent block was rewritten: %q", rendered)
	}
}

func TestRenderKeepsUnknownKeys(t *testing.T) {
	document, err := Read([]byte("<!-- lore-master\nowner: docs-team\npage-id: 1\n-->\n# Title\n"))
	if err != nil {
		t.Fatal(err)
	}
	annotation := *document.Annotation
	annotation.Version = 2
	rendered, err := Render([]byte("<!-- lore-master\nowner: docs-team\npage-id: 1\n-->\n# Title\n"), annotation)
	if err != nil {
		t.Fatal(err)
	}
	if want := "<!-- lore-master\npage-id: 1\nversion: 2\nowner: docs-team\n-->\n# Title\n"; string(rendered) != want {
		t.Fatalf("rendered %q, want %q", rendered, want)
	}
}

func TestRenderRejectsValuesThatWouldBreakTheBlock(t *testing.T) {
	cases := map[string]struct {
		annotation Annotation
		message    string
	}{
		"closing marker": {Annotation{Title: "A --> B"}, `the annotation's "title" value cannot contain a line break or "-->": "A --> B"`},
		"line break":     {Annotation{Parent: "a\nb.md"}, `the annotation's "parent" value cannot contain a line break or "-->": "a\nb.md"`},
		"known key":      {Annotation{Unknown: []Field{{Key: "page-id", Value: "1"}}}, `the annotation cannot carry "page-id" as an unknown key`},
		"bad key":        {Annotation{Unknown: []Field{{Key: "a: b", Value: "1"}}}, `the annotation cannot carry "a: b" as an unknown key`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Render([]byte("# Title\n"), tc.annotation)
			if err == nil || err.Error() != tc.message {
				t.Fatalf("error %v, want %q", err, tc.message)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte("# Title\r\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	changed, err := Write(path, Annotation{PageID: "1"})
	if err != nil || !changed {
		t.Fatalf("first write: changed=%v err=%v", changed, err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != "<!-- lore-master\r\npage-id: 1\r\n-->\r\n# Title\r\n" {
		t.Fatalf("content %q", content)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o640 {
		t.Fatalf("mode %v", info.Mode().Perm())
	}

	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	changed, err = Write(path, Annotation{PageID: "1"})
	if err != nil || changed {
		t.Fatalf("second write: changed=%v err=%v", changed, err)
	}
	if info, _ := os.Stat(path); !info.ModTime().Equal(old) {
		t.Fatal("an unchanged annotation must not touch the file")
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}
