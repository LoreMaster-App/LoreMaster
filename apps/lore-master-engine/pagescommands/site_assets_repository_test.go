package pagescommands

import (
	"os"
	"path/filepath"
	"testing"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

func parse(t *testing.T, path, content string) documentparsing.MarkdownDocument {
	t.Helper()
	document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath(path), []byte(content))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	return document
}

func TestCollectSiteAssetsGathersReferencedFiles(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("assets/logo.png", "PNGDATA")
	write("assets/clip.gif", "GIFDATA")
	write("files/spec.pdf", "PDFDATA")

	readme := parse(t, "README.md", "# Home\n\n"+
		"<p align=\"center\"><img src=\"assets/logo.png\" alt=\"logo\"></p>\n\n"+
		"![a clip](assets/clip.gif)\n\n"+
		"[the spec](files/spec.pdf)\n\n"+
		"![remote](https://example.com/x.png)\n")

	files, warnings := collectSiteAssets(root, []documentparsing.MarkdownDocument{readme})

	got := map[string]string{}
	for _, file := range files {
		got[file.Path] = string(file.Content)
	}
	for path, want := range map[string]string{
		"assets/logo.png": "PNGDATA",
		"assets/clip.gif": "GIFDATA",
		"files/spec.pdf":  "PDFDATA",
	} {
		if got[path] != want {
			t.Errorf("asset %s = %q, want %q", path, got[path], want)
		}
	}
	if _, remote := got["https://example.com/x.png"]; remote {
		t.Error("a remote image must not be published")
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestCollectSiteAssetsDeduplicatesAcrossDocuments(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "shared.png"), []byte("X"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := parse(t, "a.md", "# A\n\n![x](assets/shared.png)\n")
	b := parse(t, "b.md", "# B\n\n![x](assets/shared.png)\n")

	files, _ := collectSiteAssets(root, []documentparsing.MarkdownDocument{a, b})

	count := 0
	for _, file := range files {
		if file.Path == "assets/shared.png" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("shared asset published %d times, want 1", count)
	}
}

func TestCollectSiteAssetsWarnsOnMissingFile(t *testing.T) {
	root := t.TempDir()
	doc := parse(t, "README.md", "# Home\n\n![gone](assets/missing.gif)\n")

	files, warnings := collectSiteAssets(root, []documentparsing.MarkdownDocument{doc})

	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning for the missing asset")
	}
}
