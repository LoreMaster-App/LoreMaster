package sitepublish

import (
	"os"
	"path/filepath"
	"testing"

	"lore-master/libs/github-pages/siterender"
)

func siteOf(paths ...string) []siterender.SiteFile {
	files := make([]siterender.SiteFile, 0, len(paths))
	for _, path := range paths {
		files = append(files, siterender.SiteFile{Path: path, Content: []byte("content of " + path)})
	}

	return files
}

func put(t *testing.T, dir, name string) {
	t.Helper()
	target := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))

	return err == nil
}

func TestWriteSiteCreatesTheFolderWithNestedFilesAndTheMarker(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dist", "docs")

	if err := WriteSite(dir, siteOf("index.html", "guide/setup.html")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "guide/setup.html", MarkerFileName} {
		if !exists(dir, name) {
			t.Errorf("%s was not written", name)
		}
	}
}

func TestWriteSiteReplacesWhatAnEarlierBuildWrote(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSite(dir, siteOf("index.html", "old.html")); err != nil {
		t.Fatal(err)
	}

	if err := WriteSite(dir, siteOf("index.html")); err != nil {
		t.Fatal(err)
	}
	if exists(dir, "old.html") {
		t.Error("a page the site no longer has was left behind")
	}
	if !exists(dir, "index.html") {
		t.Error("index.html is missing")
	}
}

func TestWriteSiteRefusesAFolderWithOtherContentAndLeavesItAlone(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "index.html") // somebody's web app
	put(t, dir, "assets/app.js")

	err := WriteSite(dir, siteOf("index.html"))
	if !IsForeignFolder(err) {
		t.Fatalf("want a foreign-folder refusal, got %v", err)
	}
	if !exists(dir, "assets/app.js") || exists(dir, MarkerFileName) {
		t.Error("the refused folder was modified")
	}
}

func TestWriteSiteAcceptsASiteWrittenBeforeTheMarkerExisted(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "index.html")
	put(t, dir, "assets/lore-master.css")

	if err := WriteSite(dir, siteOf("index.html")); err != nil {
		t.Fatalf("an earlier LoreMaster site must be replaceable: %v", err)
	}
	if !exists(dir, MarkerFileName) {
		t.Error("the marker was not added")
	}
}

func TestWriteSiteKeepsTheGitDirectory(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, ".git/HEAD")

	if err := WriteSite(dir, siteOf("index.html")); err != nil {
		t.Fatal(err)
	}
	if !exists(dir, ".git/HEAD") {
		t.Error(".git was removed")
	}
}
