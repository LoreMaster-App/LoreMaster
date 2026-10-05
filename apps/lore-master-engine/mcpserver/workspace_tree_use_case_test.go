package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workspace writes the given files (path -> contents) into a fresh temp directory and
// returns its root. Shared by the workspace-aware tools' tests.
func workspace(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

// toolCall runs one tools/call against the given workspace and returns the result.
func toolCall(t *testing.T, root, name, arguments string) toolCallResult {
	t.Helper()
	frame := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + name + `","arguments":` + arguments + `}}`
	responses := runIn(t, root, frame)
	if len(responses) != 1 {
		t.Fatalf("want 1 response, got %d", len(responses))
	}
	var result toolCallResult
	decodeResult(t, responses[0], &result)

	return result
}

func previewView(t *testing.T, result toolCallResult) previewTreeView {
	t.Helper()
	raw, _ := json.Marshal(result.StructuredContent)
	var view previewTreeView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode preview view: %v", err)
	}

	return view
}

func TestPreviewTreeBuildsTheTreeTheSyncWould(t *testing.T) {
	root := workspace(t, map[string]string{
		"README.md":              "# Home\n",
		"readme.architecture.md": "# Architecture\n",
		"guide.md":               "# Guide\n",
	})
	view := previewView(t, toolCall(t, root, previewTreeToolName, "{}"))

	parents := map[string]string{}
	rules := map[string]string{}
	for _, node := range view.Nodes {
		parents[node.Path] = node.Parent
		rules[node.Path] = node.Rule
	}
	if len(view.Nodes) != 3 {
		t.Fatalf("want 3 nodes, got %d: %+v", len(view.Nodes), view.Nodes)
	}
	if parents["readme.architecture.md"] != "README.md" || rules["readme.architecture.md"] != "dotted-name" {
		t.Errorf("architecture nesting wrong: parent=%q rule=%q", parents["readme.architecture.md"], rules["readme.architecture.md"])
	}
	if parents["guide.md"] != "README.md" || rules["guide.md"] != "directory-index" {
		t.Errorf("guide nesting wrong: parent=%q rule=%q", parents["guide.md"], rules["guide.md"])
	}
	if parents["README.md"] != "" || rules["README.md"] != "selected-parent" {
		t.Errorf("README nesting wrong: parent=%q rule=%q", parents["README.md"], rules["README.md"])
	}
}

func TestWorkspaceToolsNeedAWorkspace(t *testing.T) {
	result := toolCall(t, "", previewTreeToolName, "{}")
	if !result.IsError || !strings.Contains(result.Content[0].Text, "no workspace") {
		t.Errorf("expected a no-workspace error, got: %+v", result)
	}
}
