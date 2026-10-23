package notescanning

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanFindsNotesInSourceFilesAndLeavesOutWhatASyncLeavesOut(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"src/a.ts":              "// TODO: in a\nlet x = 1\n",
		"src/b.go":              "package b\n\n// FIXME(alice): in b\n",
		"docs/guide.md":         "text\n<!-- NOTE: in markdown -->\n",
		"node_modules/dep/x.js": "// TODO: dependency\n",
		"build/out.js":          "// TODO: ignored by .gitignore\n",
		".gitignore":            "build/\n",
		"vendor/lib.ts":         "// TODO: excluded by the list\n",
		"image.png":             "// TODO: not a source file\n",
		"data.sql":              "SELECT 1; -- TODO: sql\n",
	})

	result, err := Scan(context.Background(), Options{WorkspaceRoot: root, Excludes: []string{"vendor/"}})
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, len(result.Notes))
	for i, note := range result.Notes {
		got[i] = note.Path + ":" + note.Tag
	}
	want := []string{"data.sql:TODO", "docs/guide.md:NOTE", "src/a.ts:TODO", "src/b.go:FIXME"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if result.FilesScanned != 5 {
		t.Fatalf("files scanned = %d", result.FilesScanned)
	}
}

func TestScanTakesTheTagsItIsGivenAndReadsGitignoredFilesWhenAsked(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"a.ts": "// TODO: a\n// WHY: because\n", "build/b.ts": "// WHY: built\n", ".gitignore": "build/\n"})

	result, err := Scan(context.Background(), Options{WorkspaceRoot: root, Tags: []string{"WHY"}, IncludeGitignored: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Notes) != 2 || result.Notes[0].Path != "a.ts" || result.Notes[1].Path != "build/b.ts" {
		t.Fatalf("got %+v", result.Notes)
	}
}

func TestScanSkipsBigAndBinaryFilesAndSaysSo(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"big.ts": "// TODO: big\n" + string(make([]byte, 2000)) + "x", "bin.ts": "// TODO: has a NUL \x00 byte\n", "ok.ts": "// TODO: ok\n"})

	result, err := Scan(context.Background(), Options{WorkspaceRoot: root, MaxFileBytes: 1000})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Notes) != 1 || result.Notes[0].Path != "ok.ts" || result.FilesSkipped != 2 || result.FilesScanned != 1 {
		t.Fatalf("got %+v", result)
	}
}

func TestScanStopsWhenAskedTo(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"a.ts": "// TODO: a\n"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Scan(ctx, Options{WorkspaceRoot: root}); err == nil {
		t.Fatal("a cancelled scan is an error")
	}
}
