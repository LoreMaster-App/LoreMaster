package treecommands

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/syncannotation"
)

func connect(t *testing.T) *jsonrpc2.Conn {
	t.Helper()
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodWorkspaceTree: WorkspaceTree(),
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
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// synced is body preceded by an annotation recording body's current hash.
func synced(t *testing.T, body string, pageID string) string {
	t.Helper()
	document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath("x.md"), []byte(body))
	if err != nil {
		t.Fatal(err)
	}

	return "<!-- lore-master\npage-id: " + pageID + "\nversion: 2\ncontent-hash: " + syncannotation.ContentHash(document.Body) + "\n-->\n" + body
}

const confluenceYAML = `version: 1
outputs:
  - platform: confluence
    baseUrl: https://acme.atlassian.net/wiki
    space: ENG
    parentPageId: "100"
    titlePrefix: ENG
    content:
      - type: markdown
        roots: ["."]
`

func tree(t *testing.T, conn *jsonrpc2.Conn, root string, output int) (rpcprotocol.WorkspaceTreeResult, error) {
	t.Helper()
	var result rpcprotocol.WorkspaceTreeResult
	err := conn.Call(context.Background(), rpcprotocol.MethodWorkspaceTree, rpcprotocol.WorkspaceTreeParams{WorkspaceRoot: root, Output: output}, &result)

	return result, err
}

func TestTreeNestsPagesAndClassifiesThemFromTheFilesAlone(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		workspacesettings.FileName: confluenceYAML,
		"README.md":                synced(t, "# Home\n\nWelcome.\n", "11"),
		"readme.setup.md":          "# Setup\n",
		"docs/guide.md":            synced(t, "# Guide\n", "12") + "\nan edit after the sync\n",
	})

	result, err := tree(t, connect(t), root, 0)
	if err != nil {
		t.Fatal(err)
	}
	type view struct{ Path, Title, PageTitle, Parent, Status, PageID string }
	var got []view
	for _, node := range result.Nodes {
		got = append(got, view{node.Path, node.Title, node.PageTitle, node.Parent, node.Status, node.PageID})
	}
	want := []view{
		{"README.md", "Home", "ENG: Home", "", "synced", "11"},
		{"readme.setup.md", "Setup", "ENG: Setup", "README.md", "new", ""},
		{"docs/guide.md", "Guide", "ENG: Guide", "README.md", "local-changes", "12"},
	}
	slices.SortFunc(got, func(a, b view) int { return len(a.Path) - len(b.Path) })
	slices.SortFunc(want, func(a, b view) int { return len(a.Path) - len(b.Path) })
	if !slices.Equal(got, want) {
		t.Fatalf("nodes\n got: %+v\nwant: %+v", got, want)
	}
	if len(result.Problems) != 0 {
		t.Fatalf("problems %q", result.Problems)
	}
}

func TestTreeListsParentsBeforeTheirChildren(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		workspacesettings.FileName: confluenceYAML,
		"readme.a.b.md":            "# B\n",
		"readme.a.md":              "# A\n",
		"README.md":                "# Home\n",
	})
	result, err := tree(t, connect(t), root, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, node := range result.Nodes {
		if node.Parent != "" && !seen[node.Parent] {
			t.Fatalf("%s comes before its parent %s", node.Path, node.Parent)
		}
		seen[node.Path] = true
	}
	if len(result.Nodes) != 3 {
		t.Fatalf("got %d nodes", len(result.Nodes))
	}
}

func TestTreeHonoursTheIgnoreList(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		workspacesettings.FileName: "ignore: [drafts/]\n" + confluenceYAML,
		"keep.md":                  "# Keep\n", "drafts/wip.md": "# WIP\n",
	})
	result, err := tree(t, connect(t), root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 1 || result.Nodes[0].Path != "keep.md" {
		t.Fatalf("nodes %+v", result.Nodes)
	}
}

func TestTreeOfAGitHubPagesOutputHasNoStatus(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		workspacesettings.FileName: "version: 1\noutputs:\n  - platform: github-pages\n    content:\n      - type: markdown\n        roots: [\".\"]\n",
		"README.md":                "# Home\n",
	})
	result, err := tree(t, connect(t), root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 1 || result.Nodes[0].Status != "" {
		t.Fatalf("nodes %+v", result.Nodes)
	}
}

func TestTreeRefusesBadParams(t *testing.T) {
	conn := connect(t)
	root := t.TempDir()
	write(t, root, map[string]string{workspacesettings.FileName: confluenceYAML})

	if _, err := tree(t, conn, "relative/path", 0); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("relative root: %v", err)
	}
	if _, err := tree(t, conn, root, 5); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("missing output: %v", err)
	}
	write(t, root, map[string]string{workspacesettings.FileName: "version: 1\nnonsense: true\n"})
	if _, err := tree(t, conn, root, 0); code(err) != rpcprotocol.CodeInvalidSettings {
		t.Fatalf("bad settings: %v", err)
	}
}

func TestTreeWithNoSettingsFileUsesTheDefaults(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"README.md": "# Home\n"})
	result, err := tree(t, connect(t), root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 1 || result.Nodes[0].Status != "new" {
		t.Fatalf("nodes %+v", result.Nodes)
	}
}

func TestTreeReportsTheFilesLeftOutAndTheRuleThatLeftEachOut(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		workspacesettings.FileName: "version: 1\nignore:\n  - CLAUDE.md\noutputs:\n  - platform: github-pages\n    direction: to-platform\n    content:\n      - type: markdown\n        roots: [\"docs\"]\n        excludes: [\"docs/drafts/\"]\n        template: default\n",
		"docs/README.md":           "# Docs\n",
		"docs/drafts/idea.md":      "# Idea\n",
		"CLAUDE.md":                "# Guide\n",
		"notes/other.md":           "# Other\n",
	})

	result, err := tree(t, connect(t), root, 0)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, left := range result.LeftOut {
		got[left.Path] = left.Rule + "|" + left.Pattern
	}
	want := map[string]string{
		"CLAUDE.md":           "ignore|CLAUDE.md",
		"docs/drafts/idea.md": "excludes|docs/drafts/",
		"notes/other.md":      "outside-roots|",
	}
	if !slices.Equal(sortedPairs(got), sortedPairs(want)) || result.LeftOutTotal != 3 {
		t.Fatalf("left out %v (total %d), want %v", got, result.LeftOutTotal, want)
	}
}

func sortedPairs(values map[string]string) []string {
	pairs := make([]string, 0, len(values))
	for path, rule := range values {
		pairs = append(pairs, path+"="+rule)
	}
	slices.Sort(pairs)

	return pairs
}

func TestLocalTreeShowsEveryFileWithWhereItSyncsAndWhetherGitIgnoresIt(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		workspacesettings.FileName: `version: 1
ignore: ["notes/"]
outputs:
  - platform: confluence
    baseUrl: https://acme.atlassian.net/wiki
    space: ENG
    parentPageId: "100"
    titlePrefix: ENG
    content:
      - type: markdown
        roots: ["docs"]
  - platform: github-pages
    content:
      - type: markdown
        roots: ["."]
    include: ["notes/public.md"]
`,
		".gitignore":      "scratch.md\n",
		"README.md":       "# Home\n",
		"docs/guide.md":   "# Guide\n",
		"notes/plan.md":   "# Plan\n",
		"notes/public.md": "# Public\n",
		"scratch.md":      "# Scratch\n",
	})

	var result rpcprotocol.WorkspaceTreeResult
	err := connect(t).Call(context.Background(), rpcprotocol.MethodWorkspaceTree,
		rpcprotocol.WorkspaceTreeParams{WorkspaceRoot: root, Scope: rpcprotocol.TreeScopeLocal}, &result)
	if err != nil {
		t.Fatal(err)
	}

	type view struct {
		Ignored  bool
		SyncedTo []int
	}
	got := map[string]view{}
	for _, node := range result.Nodes {
		got[node.Path] = view{node.GitIgnored, node.SyncedTo}
	}
	want := map[string]view{
		"README.md":       {false, []int{1}},
		"docs/guide.md":   {false, []int{0, 1}},
		"notes/plan.md":   {false, nil},
		"notes/public.md": {false, []int{1}},
		"scratch.md":      {true, nil},
	}
	if len(got) != len(want) {
		t.Fatalf("files %v, want %v", got, want)
	}
	for path, expected := range want {
		if actual := got[path]; actual.Ignored != expected.Ignored || !slices.Equal(actual.SyncedTo, expected.SyncedTo) {
			t.Errorf("%s: got %+v, want %+v", path, actual, expected)
		}
	}
}
