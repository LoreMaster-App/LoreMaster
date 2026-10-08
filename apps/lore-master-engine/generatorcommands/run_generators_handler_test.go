package generatorcommands

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/content-generation/generatorregistry"
	"lore-master/libs/documentation-sync/documentloading"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documenttree"
)

func connect(t *testing.T) *jsonrpc2.Conn {
	t.Helper()
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodGeneratorsRun: RunGenerators(),
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	conn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error) { return nil, nil })))
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func code(err error) int64 {
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return wire.Code
	}

	return 0
}

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for relative, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const junit = `<testsuites>
  <testsuite name="Pages view" tests="2" failures="1">
    <testcase classname="Pages view" name="nests children"/>
    <testcase classname="Pages view" name="sorts"><failure message="expected a, got b">stack</failure></testcase>
  </testsuite>
  <testsuite name="Status policy" tests="1"><testcase classname="Status" name="combines"/></testsuite>
</testsuites>`

const settingsYAML = `version: 1
generators:
  - type: test-results
    output: docs/tests
    title: CI results
  - type: test-results
    input: [ci/]
    output: docs/ci-tests
outputs:
  - platform: github-pages
    content:
      - type: markdown
        roots: ["docs"]
`

func call(t *testing.T, conn *jsonrpc2.Conn, root string, generators ...int) (rpcprotocol.GeneratorsRunResult, error) {
	t.Helper()
	var result rpcprotocol.GeneratorsRunResult
	err := conn.Call(context.Background(), rpcprotocol.MethodGeneratorsRun, rpcprotocol.GeneratorsRunParams{WorkspaceRoot: root, Generators: generators}, &result)

	return result, err
}

func TestRunWritesPagesThenLeavesThemAloneWhenNothingChanged(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{workspacesettings.FileName: settingsYAML, "reports/junit.xml": junit})
	conn := connect(t)

	first, err := call(t, conn, root, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(first.Runs) != 1 || first.Runs[0].Index != 0 || first.Runs[0].Type != "test-results" || first.Runs[0].Output != "docs/tests" || first.Runs[0].Error != "" {
		t.Fatalf("run %+v", first.Runs)
	}
	want := []string{"docs/tests/README.md", "docs/tests/pages-view.md", "docs/tests/status-policy.md"}
	if !slices.Equal(first.Runs[0].Written, want) {
		t.Fatalf("written %q, want %q", first.Runs[0].Written, want)
	}
	index, err := os.ReadFile(filepath.Join(root, "docs", "tests", "README.md"))
	if err != nil || !strings.Contains(string(index), "# CI results") || !strings.Contains(string(index), "generated: test-results") {
		t.Fatalf("index %s (%v)", index, err)
	}

	second, err := call(t, conn, root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Runs[0].Written) != 0 || !slices.Equal(second.Runs[0].Unchanged, want) {
		t.Fatalf("a rerun rewrote files: %+v", second.Runs[0])
	}
}

func TestGeneratedPagesJoinTheTreeLikeAnyOtherMarkdown(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{workspacesettings.FileName: settingsYAML, "reports/junit.xml": junit, "docs/README.md": "# Docs\n"})
	if _, err := call(t, connect(t), root, 0); err != nil {
		t.Fatal(err)
	}

	loaded, err := workspacesettings.LoadSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := documentloading.LoadOutputDocuments(context.Background(), root, loaded.Settings.Outputs[0], loaded.Settings.DiscoveryScope())
	if err != nil || len(documents.Problems) != 0 {
		t.Fatalf("load: %v %q", err, documents.Problems)
	}
	tree, err := documenttree.BuildTree(documents.Documents)
	if err != nil {
		t.Fatal(err)
	}

	parents := map[string]string{}
	tree.Walk(func(node *documenttree.TreeNode, _ int) {
		parent := ""
		if node.Decision.Parent != nil {
			parent = string(*node.Decision.Parent)
		}
		parents[string(node.Document.Path)] = parent
	})
	if parents["docs/tests/README.md"] != "docs/README.md" || parents["docs/tests/pages-view.md"] != "docs/tests/README.md" || parents["docs/tests/status-policy.md"] != "docs/tests/README.md" {
		t.Fatalf("nesting %v", parents)
	}
}

func TestRunWithNoIndexesRunsEveryGeneratorAndOneFailureDoesNotStopTheRest(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{workspacesettings.FileName: settingsYAML, "reports/junit.xml": junit, "docs/ci-tests/README.md": "# Hand-written\n"})

	result, err := call(t, connect(t), root)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Runs) != 2 || len(result.Runs[0].Written) != 3 {
		t.Fatalf("runs %+v", result.Runs)
	}
	second := result.Runs[1]
	if len(second.Warnings) == 0 || !strings.Contains(second.Warnings[0], "was not generated by test-results") {
		t.Fatalf("the hand-written page was not protected: %+v", second)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "docs", "ci-tests", "README.md")); string(content) != "# Hand-written\n" {
		t.Fatalf("a hand-written file was overwritten: %q", content)
	}
}

func TestRunRefusesBadParamsAndSettings(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	write(t, root, map[string]string{workspacesettings.FileName: settingsYAML})

	if _, err := call(t, conn, "relative"); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("relative root: %v", err)
	}
	if _, err := call(t, conn, root, 7); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("missing generator: %v", err)
	}
	write(t, root, map[string]string{workspacesettings.FileName: "version: 1\ngenerators:\n  - type: crayon\n    output: docs/x\n"})
	if _, err := call(t, conn, root); code(err) != rpcprotocol.CodeInvalidSettings || !strings.Contains(err.Error(), "crayon") {
		t.Fatalf("bad settings: %v", err)
	}
}

func TestRunWithoutGeneratorsDoesNothing(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"README.md": "# Home\n"})

	result, err := call(t, connect(t), root)
	if err != nil || result.Runs == nil || len(result.Runs) != 0 {
		t.Fatalf("result %+v, err %v", result, err)
	}
}

func TestEveryBuiltGeneratorTypeHasAGenerator(t *testing.T) {
	for _, kind := range workspacesettings.BuiltGeneratorTypes {
		if _, found := generatorregistry.For(kind); !found {
			t.Errorf("the settings accept %q but no generator is registered for it", kind)
		}
	}
	for _, kind := range generatorregistry.Types() {
		if !slices.Contains(workspacesettings.BuiltGeneratorTypes, kind) {
			t.Errorf("a generator is registered for %q but the settings refuse it", kind)
		}
	}
}

func TestRunGeneratesGoPackageDocumentation(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		workspacesettings.FileName: "version: 1\ngenerators:\n  - type: go-docs\n    input: [pkg/, '!pkg/internal/']\n    output: docs/api\n    title: API\noutputs:\n  - platform: github-pages\n    content:\n      - type: markdown\n        roots: [\"docs\"]\n",
		"go.mod":                   "module example.com/m\n\ngo 1.24\n",
		"pkg/a/a.go":               "// Package a does a.\npackage a\n\n// Do does it.\nfunc Do() {}\n",
		"pkg/internal/b/b.go":      "// Package b is hidden.\npackage b\n",
		"other/c.go":               "// Package c is not selected.\npackage c\n",
	})

	result, err := call(t, connect(t), root, 0)
	if err != nil {
		t.Fatal(err)
	}

	entry := result.Runs[0]
	if want := []string{"docs/api/README.md", "docs/api/pkg/a/README.md"}; entry.Error != "" || !slices.Equal(entry.Written, want) {
		t.Fatalf("run %+v, want %q", entry, want)
	}
	page, err := os.ReadFile(filepath.Join(root, "docs", "api", "pkg", "a", "README.md"))
	if err != nil || !strings.Contains(string(page), "# example.com/m/pkg/a") || !strings.Contains(string(page), "generated: go-docs") || !strings.Contains(string(page), "### func Do") {
		t.Fatalf("page %s (%v)", page, err)
	}
}
