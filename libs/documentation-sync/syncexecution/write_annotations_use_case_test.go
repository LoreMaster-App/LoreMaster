package syncexecution

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/syncstate"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

func writeIn(t *testing.T, root string, path string, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readIn(t *testing.T, root string, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

var technical = StateTarget{
	Key:    "Technical docs",
	Output: workspacesettings.Output{Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG"},
}

func TestWriteAnnotationsRecordsTheStateInOneFileAndLeavesTheMarkdownAlone(t *testing.T) {
	root := t.TempDir()
	writeIn(t, root, "docs/new.md", "# New\r\n\r\nWindows line endings.\r\n")
	writeIn(t, root, "untouched.md", "# Untouched\n")
	if err := os.Mkdir(filepath.Join(root, "folder.md"), 0o750); err != nil {
		t.Fatal(err)
	}
	fresh := syncannotation.Annotation{Platform: "confluence", PageID: "p9", Version: 1, ContentHash: "sha256:abc"}

	result := WriteAnnotations(root, SyncReport{Pages: []PageResult{
		{Path: "docs/new.md", Outcome: Written, Annotation: &fresh},
		{Path: "untouched.md", Outcome: Unchanged},
		{Path: "gone.md", Outcome: Written, URL: "https://docs.example/pages/p8", Annotation: &fresh},
		{Outcome: Trashed, PageID: "p3"},
	}}, technical)

	if !slices.Equal(result.Rewritten, []documentdiscovery.DocumentPath{"lore-master.json"}) {
		t.Fatalf("rewritten %q (the Markdown must not change)", result.Rewritten)
	}
	if got := readIn(t, root, "docs/new.md"); got != "# New\r\n\r\nWindows line endings.\r\n" {
		t.Fatalf("the Markdown changed: %q", got)
	}
	state, err := syncstate.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	storage := state.StorageNamed("Technical docs")
	if storage.Space != "ENG" || storage.Pages["docs/new.md"].PageID != "p9" {
		t.Fatalf("state %+v", storage)
	}
	if _, kept := storage.Pages["gone.md"]; kept {
		t.Fatal("an entry for a file that does not exist was kept")
	}
	if len(result.Warnings) != 1 || !strings.HasPrefix(result.Warnings[0], "gone.md: the page was synced (https://docs.example/pages/p8), but the file could not be updated") {
		t.Fatalf("warnings %q", result.Warnings)
	}
}

func TestAnUnchangedSyncWritesNothing(t *testing.T) {
	root := t.TempDir()
	writeIn(t, root, "a.md", "# A\n")
	page := syncannotation.Annotation{PageID: "p1", Version: 2}
	report := SyncReport{Pages: []PageResult{{Path: "a.md", Outcome: Written, Annotation: &page}}}
	WriteAnnotations(root, report, technical)

	again := WriteAnnotations(root, report, technical)

	if len(again.Rewritten) != 0 {
		t.Fatalf("a second identical write-back rewrote %q", again.Rewritten)
	}
}

func TestTwoStoragesKeepTheirOwnPagesForTheSameFile(t *testing.T) {
	root := t.TempDir()
	writeIn(t, root, "a.md", "# A\n")
	manuals := StateTarget{Key: "User manuals", Output: workspacesettings.Output{Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "DOC"}}
	one, two := syncannotation.Annotation{PageID: "p1"}, syncannotation.Annotation{PageID: "q9"}

	WriteAnnotations(root, SyncReport{Pages: []PageResult{{Path: "a.md", Outcome: Written, Annotation: &one}}}, technical)
	WriteAnnotations(root, SyncReport{Pages: []PageResult{{Path: "a.md", Outcome: Written, Annotation: &two}}}, manuals)

	state, err := syncstate.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.StorageNamed("Technical docs").Pages["a.md"].PageID != "p1" || state.StorageNamed("User manuals").Pages["a.md"].PageID != "q9" {
		t.Fatalf("state %+v", state)
	}
}

func TestTheOldAnnotationMovesIntoTheStateAndKeepsWhatTheAuthorSet(t *testing.T) {
	root := t.TempDir()
	old := syncannotation.Annotation{
		Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG",
		PageID: "p7", Version: 3, ContentHash: "sha256:abc", Title: "Custom title",
	}
	with, err := syncannotation.Render([]byte("# A\n"), old)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := syncannotation.Render([]byte("# B\n"), syncannotation.Annotation{PageID: "p8", Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	writeIn(t, root, "a.md", string(with))
	writeIn(t, root, "b.md", string(plain))
	target := technical
	target.Loaded = map[documentdiscovery.DocumentPath]*syncannotation.Annotation{"a.md": &old, "b.md": {PageID: "p8", Version: 1}}

	result := WriteAnnotations(root, SyncReport{}, target)

	if got := readIn(t, root, "a.md"); !strings.Contains(got, "title: Custom title") || strings.Contains(got, "page-id") {
		t.Fatalf("a.md should keep only the title:\n%s", got)
	}
	if got := readIn(t, root, "b.md"); got != "# B\n" {
		t.Fatalf("b.md should lose its annotation entirely:\n%q", got)
	}
	state, err := syncstate.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	pages := state.StorageNamed("Technical docs").Pages
	if pages["a.md"].PageID != "p7" || pages["a.md"].Version != 3 || pages["b.md"].PageID != "p8" {
		t.Fatalf("state %+v", pages)
	}
	if len(result.Rewritten) != 3 {
		t.Fatalf("rewritten %q", result.Rewritten)
	}
}

func TestAFileSyncedToAnotherTargetKeepsItsOwnAnnotation(t *testing.T) {
	root := t.TempDir()
	foreign := syncannotation.Annotation{Platform: "confluence", BaseURL: "https://other.atlassian.net/wiki", Space: "OPS", PageID: "x1", Version: 4}
	content, err := syncannotation.Render([]byte("# A\n"), foreign)
	if err != nil {
		t.Fatal(err)
	}
	writeIn(t, root, "a.md", string(content))
	fresh := syncannotation.Annotation{PageID: "p1"}
	target := technical
	target.Loaded = map[documentdiscovery.DocumentPath]*syncannotation.Annotation{"a.md": &foreign}

	WriteAnnotations(root, SyncReport{Pages: []PageResult{{Path: "a.md", Outcome: Written, Annotation: &fresh}}}, target)

	if got := readIn(t, root, "a.md"); !strings.Contains(got, "page-id: x1") {
		t.Fatalf("the other target's state was lost:\n%s", got)
	}
}

func TestAnUnreadableStateFileIsNeverOverwritten(t *testing.T) {
	root := t.TempDir()
	writeIn(t, root, "lore-master.json", "{ not json")
	page := syncannotation.Annotation{PageID: "p1"}
	writeIn(t, root, "a.md", "# A\n")

	result := WriteAnnotations(root, SyncReport{Pages: []PageResult{{Path: "a.md", Outcome: Written, Annotation: &page}}}, technical)

	if got := readIn(t, root, "lore-master.json"); got != "{ not json" {
		t.Fatalf("the file was overwritten: %q", got)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "not valid JSON") {
		t.Fatalf("warnings %q", result.Warnings)
	}
}

func TestAPulledBodyIsWrittenWithoutTheState(t *testing.T) {
	root := t.TempDir()
	writeIn(t, root, "a.md", "# Old\n")
	page := syncannotation.Annotation{PageID: "p1", Version: 5}

	WriteAnnotations(root, SyncReport{Pages: []PageResult{{Path: "a.md", Outcome: Written, Annotation: &page, PulledBody: []byte("# Pulled\n")}}}, technical)

	if got := readIn(t, root, "a.md"); got != "# Pulled\n" {
		t.Fatalf("a.md:\n%q", got)
	}
}
