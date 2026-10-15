package syncstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/syncannotation"
)

func TestApplyGivesEachDocumentItsRecordedPageAndKeepsWhatTheAuthorSet(t *testing.T) {
	root := t.TempDir()
	state := Empty()
	state.Storages["Technical docs"] = Storage{
		Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG",
		Pages: map[string]Page{"a.md": {PageID: "p1", Version: 4, ContentHash: "sha256:x", SyncedAt: "2026-10-01T12:00:00Z", Attachments: map[string]string{"d.svg": "sha256:y"}}},
	}
	if _, err := Write(root, state); err != nil {
		t.Fatal(err)
	}
	documents := []documentparsing.MarkdownDocument{
		{Path: "a.md", Annotation: &syncannotation.Annotation{Title: "Custom"}},
		{Path: "b.md"},
	}

	applied := Apply(root, "Technical docs", documents)

	got := documents[0].Annotation
	if got == nil || got.PageID != "p1" || got.Version != 4 || got.Space != "ENG" || got.Title != "Custom" || got.Attachments["d.svg"] != "sha256:y" {
		t.Fatalf("annotation %+v", got)
	}
	if !got.SyncedAt.Equal(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("synced at %v", got.SyncedAt)
	}
	if documents[1].Annotation != nil {
		t.Fatalf("a file the state does not know got %+v", documents[1].Annotation)
	}
	if applied.Loaded["a.md"].PageID != "" || applied.Loaded["a.md"].Title != "Custom" {
		t.Fatalf("loaded must be the file's own annotation: %+v", applied.Loaded["a.md"])
	}
}

func TestAnotherStoragesStateIsNotApplied(t *testing.T) {
	root := t.TempDir()
	state := Empty()
	state.Storages["User manuals"] = Storage{Pages: map[string]Page{"a.md": {PageID: "q1"}}}
	if _, err := Write(root, state); err != nil {
		t.Fatal(err)
	}
	documents := []documentparsing.MarkdownDocument{{Path: "a.md"}}

	Apply(root, "Technical docs", documents)

	if documents[0].Annotation != nil {
		t.Fatalf("got %+v", documents[0].Annotation)
	}
}

func TestAnOldAnnotationStillAppliesUntilTheFirstMigratingSync(t *testing.T) {
	root := t.TempDir()
	documents := []documentparsing.MarkdownDocument{{Path: "a.md", Annotation: &syncannotation.Annotation{PageID: "p9", Version: 2}}}

	Apply(root, "Technical docs", documents)

	if documents[0].Annotation.PageID != "p9" {
		t.Fatalf("got %+v", documents[0].Annotation)
	}
}

func TestTheStateFileIsStableReadableJSON(t *testing.T) {
	root := t.TempDir()
	state := Empty()
	state.Storages["B"] = Storage{Pages: map[string]Page{"z.md": {PageID: "2"}, "a.md": {PageID: "1"}}}
	state.Storages["A"] = Storage{Pages: map[string]Page{"a.md": {PageID: "3"}}}

	changed, err := Write(root, state)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	again, err := Write(root, state)
	if err != nil || again {
		t.Fatalf("an identical write changed the file (%v, %v)", again, err)
	}
	content, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Index(text, `"A"`) > strings.Index(text, `"B"`) || strings.Index(text, `"a.md"`) > strings.Index(text, `"z.md"`) || !strings.HasSuffix(text, "}\n") {
		t.Fatalf("not stable:\n%s", text)
	}
}

func TestAnEmptyStateCreatesNoFile(t *testing.T) {
	root := t.TempDir()

	changed, err := Write(root, Empty())

	if err != nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if _, statErr := os.Stat(filepath.Join(root, FileName)); statErr == nil {
		t.Fatal("an empty state created a file")
	}
}

func TestANewerLayoutIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(`{"version": 99, "storages": {}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Read(root); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("err %v", err)
	}
}
