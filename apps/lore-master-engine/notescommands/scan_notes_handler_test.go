package notescommands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

func scan(t *testing.T, params rpcprotocol.NotesScanParams) (rpcprotocol.NotesScanResult, error) {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ScanNotes()(context.Background(), rpcserver.Call{Params: raw})
	if err != nil {
		return rpcprotocol.NotesScanResult{}, err
	}

	return result.(rpcprotocol.NotesScanResult), nil
}

func TestScanNotesWorksInAWorkspaceWithNoSettingsAndReadsTheIgnoreList(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"src/a.ts":              "// TODO(#3): first\n",
		"build/out.ts":          "// TODO: generated\n",
		".gitignore":            "build/\n",
		".vscode/settings.json": "{ \"loreMaster.ignore\": [\"legacy/\"] }\n",
		"legacy/old.ts":         "// FIXME: ignored by the list\n",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	result, err := scan(t, rpcprotocol.NotesScanParams{WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Notes) != 1 || result.Notes[0].Path != "src/a.ts" || result.Notes[0].Issue != "3" || result.Notes[0].Text != "first" {
		t.Fatalf("got %+v", result.Notes)
	}
	if len(result.Tags) == 0 {
		t.Fatal("the answer names the tags it looked for")
	}
}

func TestScanNotesRefusesARelativeRootAndTakesGivenTags(t *testing.T) {
	if _, err := scan(t, rpcprotocol.NotesScanParams{WorkspaceRoot: "relative/path"}); err == nil {
		t.Fatal("a relative workspace root is refused")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("// WHY: reasons\n// TODO: not asked for\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := scan(t, rpcprotocol.NotesScanParams{WorkspaceRoot: root, Tags: []string{"WHY"}})
	if err != nil || len(result.Notes) != 1 || result.Notes[0].Tag != "WHY" || len(result.Tags) != 1 {
		t.Fatalf("got %+v %v", result, err)
	}
}
