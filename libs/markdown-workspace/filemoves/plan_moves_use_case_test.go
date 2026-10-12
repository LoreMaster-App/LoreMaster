package filemoves

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPlanWorkspaceMovesReadsParentLinesFromDisk(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"docs/guide.md":       "# Guide\n",
		"docs/guide.setup.md": "# Setup\n",
		"docs/other.md":       "<!-- lore-master\ntitle: Other\nparent: ./guide.md\n-->\n# Other\n",
	}
	for name, content := range files {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	plan, err := PlanWorkspaceMoves(context.Background(), root, []Move{{From: "docs/guide.md", To: "docs/manual.md"}})
	if err != nil {
		t.Fatal(err)
	}

	wantRenames := []Rename{{From: "docs/guide.setup.md", To: "docs/manual.setup.md"}}
	wantEdits := []LineEdit{{Path: "docs/other.md", Line: 2, Text: "parent: ./manual.md"}}
	if !reflect.DeepEqual(plan.Renames, wantRenames) || !reflect.DeepEqual(plan.Edits, wantEdits) {
		t.Fatalf("plan = %+v, want renames %+v edits %+v", plan, wantRenames, wantEdits)
	}
}
