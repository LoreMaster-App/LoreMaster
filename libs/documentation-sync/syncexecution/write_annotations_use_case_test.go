package syncexecution

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

func TestWriteAnnotationsTouchesOnlyWhatSynced(t *testing.T) {
	root := t.TempDir()
	write := func(path string, content string) {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	synced := syncannotation.Annotation{PageID: "p7", Version: 3, ContentHash: syncannotation.ContentHash([]byte("# Same\n"))}
	same, err := syncannotation.Render([]byte("# Same\n"), synced)
	if err != nil {
		t.Fatal(err)
	}
	write("docs/new.md", "# New\r\n\r\nWindows line endings.\r\n")
	write("same.md", string(same))
	write("untouched.md", "# Untouched\n")
	if err := os.Mkdir(filepath.Join(root, "folder.md"), 0o750); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, path := range []string{"docs/new.md", "same.md", "untouched.md"} {
		if err := os.Chtimes(filepath.Join(root, filepath.FromSlash(path)), old, old); err != nil {
			t.Fatal(err)
		}
	}

	fresh := syncannotation.Annotation{Platform: "confluence", PageID: "p9", Version: 1, ContentHash: syncannotation.ContentHash([]byte("# New\r\n\r\nWindows line endings.\r\n"))}
	result := WriteAnnotations(root, SyncReport{Pages: []PageResult{
		{Path: "docs/new.md", Outcome: Written, Annotation: &fresh},
		{Path: "same.md", Outcome: Written, Annotation: &synced},
		{Path: "untouched.md", Outcome: Unchanged},
		{Path: "gone.md", Outcome: Written, URL: "https://docs.example/pages/p8", Annotation: &fresh},
		{Path: "folder.md", Outcome: Written, Annotation: &fresh},
		{Outcome: Trashed, PageID: "p3"},
	}})

	if !slices.Equal(result.Rewritten, []documentdiscovery.DocumentPath{"docs/new.md"}) {
		t.Fatalf("rewritten %q", result.Rewritten)
	}
	if len(result.Warnings) != 2 || !strings.HasPrefix(result.Warnings[0], "gone.md: the page was synced (https://docs.example/pages/p8), but the file could not be updated") ||
		!strings.HasPrefix(result.Warnings[1], "folder.md: ") {
		t.Fatalf("warnings %q", result.Warnings)
	}
	for _, path := range []string{"same.md", "untouched.md"} {
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil || !info.ModTime().Equal(old) {
			t.Errorf("%s was touched", path)
		}
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "new.md"))
	if err != nil {
		t.Fatal(err)
	}
	read, err := syncannotation.Read(content)
	if err != nil || read.Annotation == nil || read.Annotation.PageID != "p9" {
		t.Fatalf("annotation %+v %v", read.Annotation, err)
	}
	if syncannotation.ContentHash(read.Body) != fresh.ContentHash || !strings.Contains(string(content), "\r\n") {
		t.Fatalf("the body and its line endings are kept:\n%q", content)
	}
}
