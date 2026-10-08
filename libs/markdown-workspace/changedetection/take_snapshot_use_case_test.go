package changedetection

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func put(t *testing.T, root string, relative string, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func take(t *testing.T, root string, previous Snapshot) Snapshot {
	t.Helper()
	snapshot, err := Take(context.Background(), root, previous)
	if err != nil {
		t.Fatal(err)
	}

	return snapshot
}

func paths(snapshot Snapshot) []string {
	var out []string
	for path := range snapshot.Files {
		out = append(out, path)
	}
	slices.Sort(out)

	return out
}

func TestOnlyFilesThatCanMatterAreLookedAt(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{
		"README.md", "docs/a.md", ".lore-master.yaml", "src/app.ts", "svc/main.py", "reports/junit.xml",
		"node_modules/x/index.js", ".git/config", "bin/Debug/a.json", "assets/logo.png", ".hidden/a.md", "obj/b.cs", "testdata/c.md", ".eslintrc.json",
	} {
		put(t, root, file, "x")
	}

	got := paths(take(t, root, Snapshot{}))

	want := []string{".lore-master.yaml", "README.md", "docs/a.md", "reports/junit.xml", "src/app.ts", "svc/main.py"}
	if !slices.Equal(got, want) {
		t.Fatalf("looked at %v, want %v", got, want)
	}
}

func TestAnEditAnAdditionAndARemovalAreChanges(t *testing.T) {
	root := t.TempDir()
	put(t, root, "a.md", "one")
	put(t, root, "b.md", "two")
	put(t, root, "gone.md", "bye")
	before := take(t, root, Snapshot{})

	put(t, root, "a.md", "one, edited")
	put(t, root, "new.md", "hi")
	if err := os.Remove(filepath.Join(root, "gone.md")); err != nil {
		t.Fatal(err)
	}
	after := take(t, root, before)

	if got := Changed(before, after); !slices.Equal(got, []string{"a.md", "gone.md", "new.md"}) {
		t.Fatalf("changed %v", got)
	}
}

func TestATouchedFileWithTheSameContentIsNotChanged(t *testing.T) {
	root := t.TempDir()
	put(t, root, "a.md", "same")
	before := take(t, root, Snapshot{})

	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(root, "a.md"), later, later); err != nil {
		t.Fatal(err)
	}
	after := take(t, root, before)

	if got := Changed(before, after); len(got) != 0 {
		t.Fatalf("changed %v", got)
	}
	if after.Files["a.md"].ModTime == before.Files["a.md"].ModTime {
		t.Fatal("the new modification time was not recorded")
	}
}

func TestAnUnchangedFileIsNotReadAgain(t *testing.T) {
	root := t.TempDir()
	put(t, root, "a.md", "content")
	before := take(t, root, Snapshot{})
	// A hash the file's content cannot have: it is kept only if the file is not read again.
	state := before.Files["a.md"]
	state.Hash = "kept"
	before.Files["a.md"] = state

	after := take(t, root, before)

	if after.Files["a.md"].Hash != "kept" {
		t.Fatalf("the file was read again: %q", after.Files["a.md"].Hash)
	}
}

func TestACancelledLookStops(t *testing.T) {
	root := t.TempDir()
	put(t, root, "a.md", "x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Take(ctx, root, Snapshot{}); err == nil {
		t.Fatal("expected the cancellation")
	}
}

func TestWatchedFileNames(t *testing.T) {
	cases := map[string]bool{
		"a.md": true, "A.MD": true, "dir/x.dart": true, "pubspec.yaml": true, "src/Shop.csproj": true,
		".lore-master.yaml": true, "pic.png": false, ".env": false, "dir/.hidden.md": false, "Makefile": false,
	}
	for name, want := range cases {
		if got := IsWatchedFile(name); got != want {
			t.Errorf("IsWatchedFile(%q) = %v, want %v", name, got, want)
		}
	}
}
