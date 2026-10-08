package documentloading

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
)

func workspaceWith(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func titles(loaded Loaded) []string {
	out := make([]string, len(loaded.Documents))
	for i, document := range loaded.Documents {
		out[i] = string(document.Path) + "=" + document.Title
	}

	return out
}

func markdownOutput(roots []string, excludes ...string) workspacesettings.Output {
	return workspacesettings.Output{Content: []workspacesettings.Content{{Type: "markdown", Roots: roots, Excludes: excludes}}}
}

var skipGitignored = workspacesettings.DiscoveryScope{SkipGitignored: true}

func TestLoadParsesEveryMarkdownFileTheOutputCovers(t *testing.T) {
	root := workspaceWith(t, map[string]string{
		"README.md": "# Home\n", "docs/guide.md": "# Guide\n", "notes.txt": "not markdown",
	})
	loaded, err := LoadOutputDocuments(context.Background(), root, markdownOutput(nil), skipGitignored)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"README.md=Home", "docs/guide.md=Guide"}; !slices.Equal(titles(loaded), want) {
		t.Fatalf("got %q, want %q", titles(loaded), want)
	}
	if len(loaded.Problems) != 0 {
		t.Fatalf("problems %q", loaded.Problems)
	}
}

func TestLoadMergesEntriesWithoutRepeatsAndSkipsOtherTypes(t *testing.T) {
	root := workspaceWith(t, map[string]string{"docs/a.md": "# A\n", "docs/b.md": "# B\n"})
	output := workspacesettings.Output{Content: []workspacesettings.Content{
		{Type: "markdown", Roots: []string{"docs"}},
		{Type: "markdown", Roots: []string{"docs/a.md/.."}},
		{Type: "test-results", Roots: []string{"nowhere"}},
	}}
	loaded, err := LoadOutputDocuments(context.Background(), root, output, skipGitignored)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"docs/a.md=A", "docs/b.md=B"}; !slices.Equal(titles(loaded), want) {
		t.Fatalf("got %q, want %q", titles(loaded), want)
	}
}

func TestLoadAppliesTheScopeAndTheEntryExcludes(t *testing.T) {
	root := workspaceWith(t, map[string]string{
		".gitignore": "ignored.md\n", "ignored.md": "# Ignored\n", "kept.md": "# Kept\n",
		"drafts/wip.md": "# Draft\n", "internal/x.md": "# Internal\n",
	})
	output := markdownOutput(nil, "internal/")
	scope := workspacesettings.DiscoveryScope{SkipGitignored: true, Ignore: []string{"drafts/"}}

	loaded, err := LoadOutputDocuments(context.Background(), root, output, scope)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"kept.md=Kept"}; !slices.Equal(titles(loaded), want) {
		t.Fatalf("got %q, want %q", titles(loaded), want)
	}

	scope.SkipGitignored = false
	loaded, err = LoadOutputDocuments(context.Background(), root, output, scope)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"ignored.md=Ignored", "kept.md=Kept"}; !slices.Equal(titles(loaded), want) {
		t.Fatalf("with skipGitignored off: got %q, want %q", titles(loaded), want)
	}
}

func TestLoadFailsWhenARootDoesNotExist(t *testing.T) {
	_, err := LoadOutputDocuments(context.Background(), t.TempDir(), markdownOutput([]string{"nowhere"}), skipGitignored)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestReadInWorkspaceRefusesPathsOutsideTheWorkspace(t *testing.T) {
	if _, err := readInWorkspace(t.TempDir(), documentdiscovery.DocumentPath("../escape.md")); err == nil {
		t.Fatal("expected a refusal")
	}
}
