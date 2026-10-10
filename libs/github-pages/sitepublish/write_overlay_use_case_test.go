package sitepublish

import (
	"os"
	"path/filepath"
	"testing"

	"lore-master/libs/github-pages/siterender"
)

func TestWriteOverlayKeepsPagesItDidNotWriteAndRemovesItsOwnStaleOnes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Hand-written.md"), []byte("by a person"), 0o644); err != nil {
		t.Fatal(err)
	}

	first := []siterender.SiteFile{{Path: "Home.md", Content: []byte("v1")}, {Path: "Old.md", Content: []byte("old")}}
	if err := WriteOverlay(dir, first); err != nil {
		t.Fatal(err)
	}
	second := []siterender.SiteFile{{Path: "Home.md", Content: []byte("v2")}}
	if err := WriteOverlay(dir, second); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "Hand-written.md")); err != nil {
		t.Errorf("a page LoreMaster did not write must survive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Old.md")); !os.IsNotExist(err) {
		t.Errorf("a stale page it wrote earlier must be removed, got %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "Home.md")); string(got) != "v2" {
		t.Errorf("Home.md = %q", got)
	}
}

func TestWriteOverlayNeverDeletesOutsideTheFolderWhateverTheMarkerSays(t *testing.T) {
	parent := t.TempDir()
	victim := filepath.Join(parent, "victim.txt")
	if err := os.WriteFile(victim, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(parent, "wiki")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, OverlayMarkerFileName), []byte("../victim.txt\n.git/config\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteOverlay(dir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("a hand-edited marker deleted a file outside the folder: %v", err)
	}
}
