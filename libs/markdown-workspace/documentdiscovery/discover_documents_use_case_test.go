package documentdiscovery

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// The tree is written per test rather than committed under testdata/: a committed
// .gitignore would change what this repository itself tracks, the repository's own
// .gitignore swallows node_modules and dist, and case-colliding names cannot be
// checked out on Windows or macOS at all.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		osPath := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(osPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(osPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func paths(documents []DocumentPath) []string {
	out := make([]string, len(documents))
	for i, document := range documents {
		out[i] = string(document)
	}

	return out
}

func TestDiscoverDocuments(t *testing.T) {
	tree := map[string]string{
		"README.md":                    "# Root",
		"readme.architecture.md":       "# Architecture",
		"notes.txt":                    "not markdown",
		"UPPER.MD":                     "# Upper-case extension",
		"docs/guide.md":                "# Guide",
		"docs/drafts/wip.md":           "# Draft",
		"docs/drafts/keep.md":          "# Git never re-includes a file under an excluded directory",
		"docs/private/secret.md":       "# Secret",
		"docs/.gitignore":              "drafts/\n!drafts/keep.md\n*.tmp.md\n!keep.tmp.md\n",
		"docs/scratch.tmp.md":          "# Scratch",
		"docs/keep.tmp.md":             "# Kept by negation",
		".gitignore":                   "/build/\n",
		"build/out.md":                 "# Build output",
		"src/build/still-here.md":      "# Anchored pattern only matches at the root",
		"node_modules/pkg/README.md":   "# Dependency",
		"packages/a/node_modules/x.md": "# Nested dependency",
		".git/info.md":                 "# VCS",
		"dist/api.md":                  "# Dist",
		"apps/x/out-tsc/y.md":          "# Out tsc",
		"coverage/report.md":           "# Coverage",
		".venv/lib/z.md":               "# Venv",
		"packages/a/README.md":         "# Package",
		"packages/a/CHANGELOG.md":      "# Changelog",
	}

	cases := []struct {
		name    string
		options Options
		want    []string
	}{
		{
			name: "whole workspace honours default excludes and nested .gitignore files",
			want: []string{
				"README.md", "UPPER.MD", "docs/guide.md", "docs/keep.tmp.md", "docs/private/secret.md",
				"packages/a/CHANGELOG.md", "packages/a/README.md", "readme.architecture.md", "src/build/still-here.md",
			},
		},
		{
			name:    "user excludes use gitignore syntax against workspace-relative paths",
			options: Options{Excludes: []string{"docs/private", "CHANGELOG.md"}},
			want: []string{
				"README.md", "UPPER.MD", "docs/guide.md", "docs/keep.tmp.md",
				"packages/a/README.md", "readme.architecture.md", "src/build/still-here.md",
			},
		},
		{
			name:    "roots narrow the scan and overlapping roots do not duplicate",
			options: Options{Roots: []string{"docs", "./docs/private", "packages/a/"}},
			want:    []string{"docs/guide.md", "docs/keep.tmp.md", "docs/private/secret.md", "packages/a/CHANGELOG.md", "packages/a/README.md"},
		},
		{
			name:    "a .gitignore above the scanned root still applies",
			options: Options{Roots: []string{"build", "docs/drafts"}},
			want:    []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			options := tc.options
			options.WorkspaceRoot = writeTree(t, tree)
			got, err := DiscoverDocuments(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(paths(got.Documents), tc.want) {
				t.Fatalf("documents\n got: %q\nwant: %q", paths(got.Documents), tc.want)
			}
			if len(got.Warnings) != 0 {
				t.Fatalf("unexpected warnings: %q", got.Warnings)
			}
		})
	}
}

func TestDiscoverDocumentsRejectsRootsOutsideTheWorkspace(t *testing.T) {
	for _, root := range []string{"..", "../sibling", "docs/../..", "/etc"} {
		_, err := DiscoverDocuments(context.Background(), Options{WorkspaceRoot: t.TempDir(), Roots: []string{root}})
		if err == nil || !strings.Contains(err.Error(), "inside the workspace") {
			t.Fatalf("root %q: got %v", root, err)
		}
	}
}

func TestDiscoverDocumentsFailsOnAMissingRoot(t *testing.T) {
	_, err := DiscoverDocuments(context.Background(), Options{WorkspaceRoot: t.TempDir(), Roots: []string{"nowhere"}})
	if err == nil {
		t.Fatal("expected an error for a root that does not exist")
	}
}

func TestDiscoverDocumentsStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := DiscoverDocuments(ctx, Options{WorkspaceRoot: writeTree(t, map[string]string{"a.md": "# A"})})
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("got %v", err)
	}
}

func TestDiscoverDocumentsSkipsSymbolicLinks(t *testing.T) {
	workspace := writeTree(t, map[string]string{"real.md": "# Real", "elsewhere/outside.md": "# Outside"})
	links := map[string]string{"linked.md": "real.md", "linked-dir": "elsewhere"}
	for link, target := range links {
		if err := os.Symlink(target, filepath.Join(workspace, link)); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("creating symbolic links needs developer mode on Windows: %v", err)
			}
			t.Fatal(err)
		}
	}

	got, err := DiscoverDocuments(context.Background(), Options{WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"elsewhere/outside.md", "real.md"}; !slices.Equal(paths(got.Documents), want) {
		t.Fatalf("documents %q, want %q", paths(got.Documents), want)
	}
	if want := []string{"linked.md is a symbolic link and was skipped"}; !slices.Equal(got.Warnings, want) {
		t.Fatalf("warnings %q, want %q", got.Warnings, want)
	}
}

func TestDiscoverDocumentsWarnsOnCaseCollisions(t *testing.T) {
	workspace := writeTree(t, map[string]string{"Guide.md": "# A", "guide.md": "# B", "other.md": "# C"})
	if entries, _ := os.ReadDir(workspace); len(entries) < 3 {
		t.Skip("this file system is case-insensitive, so the collision cannot be created")
	}

	got, err := DiscoverDocuments(context.Background(), Options{WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Guide.md, guide.md differ only by case and collide on Windows and macOS"}; !slices.Equal(got.Warnings, want) {
		t.Fatalf("warnings %q, want %q", got.Warnings, want)
	}
}

func TestLookup(t *testing.T) {
	discovery := NewDiscovery([]DocumentPath{"Guide.md", "guide.md", "docs/Setup.md"}, nil)
	cases := []struct {
		path   string
		want   DocumentPath
		wantOK bool
	}{
		{"guide.md", "guide.md", true},
		{"Guide.md", "Guide.md", true},
		{"docs/setup.md", "docs/Setup.md", true},
		{"GUIDE.md", "", false},
		{"missing.md", "", false},
	}
	for _, tc := range cases {
		got, ok := discovery.Lookup(tc.path)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("Lookup(%q) = %q, %v; want %q, %v", tc.path, got, ok, tc.want, tc.wantOK)
		}
	}
}
