package notesdocs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func workspace(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func generate(t *testing.T, root string, spec generatedfile.Spec) string {
	t.Helper()
	output, err := Generate(context.Background(), root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Files) != 1 || output.Files[0].Path != "README.md" {
		t.Fatalf("files %+v", output.Files)
	}

	return string(output.Files[0].Body)
}

func TestTheNotesBecomeOnePageWithATablePerTag(t *testing.T) {
	root := workspace(t, map[string]string{
		"src/a.ts": "// TODO(alice, #12, 2026-12-01): handle the empty list\n// FIXME: a | b <c>\n",
		"src/b.go": "package b\n\n// TODO: second todo\n// HACK: retry\n",
		"doc.md":   "<!-- NOTE: generated page -->\n",
	})

	page := generate(t, root, generatedfile.Spec{})

	for _, want := range []string{
		"# Notes\n",
		"5 in 3 files",
		"| TODO | 2 |",
		"## TODO (2)",
		"| Note | Where | Owner | Issue | Due |",
		"| handle the empty list | `src/a.ts:1` | alice | #12 | 2026-12-01 |",
		"| second todo | `src/b.go:3` |  |  |  |",
		"## FIXME (1)\n\n| Note | Where |\n| --- | --- |\n| a \\| b &lt;c&gt; | `src/a.ts:2` |",
		"## NOTE (1)",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q in:\n%s", want, page)
		}
	}
	if strings.Index(page, "## TODO") > strings.Index(page, "## FIXME") || strings.Index(page, "## FIXME") > strings.Index(page, "## HACK") {
		t.Errorf("tags follow the default order:\n%s", page)
	}
}

func TestTheSameNotesGiveTheSamePageAndInputSelectsTheFiles(t *testing.T) {
	root := workspace(t, map[string]string{"src/a.ts": "// TODO: in src\n", "vendor/x.ts": "// TODO: in vendor\n"})

	first := generate(t, root, generatedfile.Spec{Input: []string{"src/"}, Title: "Open work"})
	second := generate(t, root, generatedfile.Spec{Input: []string{"src/"}, Title: "Open work"})

	if first != second || !strings.HasPrefix(first, "# Open work\n") || strings.Contains(first, "vendor") || !strings.Contains(first, "in src") {
		t.Fatalf("got:\n%s", first)
	}
}

func TestWithNoNotesThePageSaysSo(t *testing.T) {
	page := generate(t, workspace(t, map[string]string{"a.ts": "let x = 1\n"}), generatedfile.Spec{})

	if !strings.Contains(page, "No notes were found") {
		t.Fatalf("got:\n%s", page)
	}
}
