package godocs

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fixtureWorkspace copies the fixture module into a fresh folder, restoring the go.mod files
// stored as go.mod.fixture: a real go.mod inside the repository would be taken for a project
// of its own by the build tooling.
func fixtureWorkspace(t *testing.T) string {
	t.Helper()
	const source = "testdata/module"
	root := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, strings.TrimSuffix(relative, ".fixture"))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}

	return root
}

func foldersOf(t *testing.T, root string, patterns ...string) map[string]string {
	t.Helper()
	folders, err := findFolders(context.Background(), root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]string{}
	for _, folder := range folders {
		found[folder.Dir] = folder.ImportPath
	}

	return found
}

func TestFindFoldersListsSourceFoldersWithTheirImportPaths(t *testing.T) {
	got := foldersOf(t, fixtureWorkspace(t))

	want := map[string]string{
		".":                "example.com/app",
		"broken":           "example.com/app/broken",
		"cmd/tool":         "example.com/app/cmd/tool",
		"empty":            "example.com/app/empty",
		"internal/secret":  "example.com/app/internal/secret",
		"mixed":            "example.com/app/mixed",
		"platform":         "example.com/app/platform",
		"scratch":          "example.com/app/scratch",
		"store":            "example.com/app/store",
		"tools/ext":        "example.com/tools/ext",
	}
	if len(got) != len(want) {
		t.Fatalf("folders %v, want %v", got, want)
	}
	for dir, importPath := range want {
		if got[dir] != importPath {
			t.Errorf("%s: import path %q, want %q", dir, got[dir], importPath)
		}
	}
}

func TestFindFoldersNeverEntersDependenciesTestDataOrHiddenFolders(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"node_modules/x", "vendor/y", "testdata/z", ".hidden/a", "_private/b", "dist/c", "real"} {
		full := filepath.Join(root, filepath.FromSlash(dir))
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(full, "x.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := foldersOf(t, root)

	if len(got) != 1 || got["real"] != "real" {
		t.Fatalf("folders %v", got)
	}
}

func TestFindFoldersIgnoresTestFilesAndOtherFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a_test.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("# n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := foldersOf(t, root); len(got) != 0 {
		t.Fatalf("folders %v", got)
	}
}

func TestFindFoldersFollowsTheInputPatterns(t *testing.T) {
	selected := foldersOf(t, fixtureWorkspace(t), "store/", "cmd/")
	if len(selected) != 2 || selected["store"] == "" || selected["cmd/tool"] == "" {
		t.Fatalf("selected %v", selected)
	}

	left := foldersOf(t, fixtureWorkspace(t), "!internal/", "!tools/", "!broken/", "!mixed/", "!platform/", "!scratch/", "!empty/")
	var dirs []string
	for dir := range left {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	if strings.Join(dirs, ",") != ".,cmd/tool,store" {
		t.Fatalf("with exclusions only, every other folder is kept: %v", dirs)
	}
}

func TestFindFoldersWithoutAGoModUsesTheFolderAsTheImportPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg", "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "a", "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := foldersOf(t, root); got["pkg/a"] != "pkg/a" {
		t.Fatalf("folders %v", got)
	}
}

func TestFindFoldersStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := findFolders(ctx, fixtureWorkspace(t), nil); err == nil {
		t.Fatal("expected the cancellation")
	}
}
