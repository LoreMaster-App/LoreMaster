package watchcommands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

const settings = `version: 1
outputs:
  - platform: github-pages
generators:
  - type: go-docs
    input: [libs/]
    output: docs/api
  - type: test-results
    output: docs/tests
`

func route(t *testing.T, root string, changed ...string) (rpcprotocol.WatchRouteResult, error) {
	t.Helper()
	params, _ := json.Marshal(rpcprotocol.WatchRouteParams{WorkspaceRoot: root, Changed: changed})
	result, err := RouteChanges()(context.Background(), rpcserver.Call{Params: params})
	if err != nil {
		return rpcprotocol.WatchRouteResult{}, err
	}

	return result.(rpcprotocol.WatchRouteResult), nil
}

func workspace(t *testing.T, yaml string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".lore-master.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestRouteNamesTheGeneratorsAndPagesAChangeCallsFor(t *testing.T) {
	root := workspace(t, settings)

	result, err := route(t, root, "libs/core/a.go", "docs/guide.md", "reports/junit.xml", "README.md")

	if err != nil {
		t.Fatal(err)
	}
	if result.Everything || !slices.Equal(result.Generators, []int{0, 1}) || !slices.Equal(result.Markdown, []string{"README.md", "docs/guide.md"}) {
		t.Fatalf("result %+v", result)
	}
}

func TestRouteAnswersEmptyListsNotNulls(t *testing.T) {
	result, err := route(t, workspace(t, settings), "image.png")

	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	if string(encoded) != `{"everything":false,"generators":[],"markdown":[]}` {
		t.Fatalf("%s", encoded)
	}
}

func TestRouteOfTheSettingsFileIsEverything(t *testing.T) {
	result, err := route(t, workspace(t, settings), ".lore-master.yaml")

	if err != nil || !result.Everything || !slices.Equal(result.Generators, []int{0, 1}) {
		t.Fatalf("result %+v err %v", result, err)
	}
}

func TestRouteRefusesARelativeWorkspaceAndInvalidSettings(t *testing.T) {
	if _, err := route(t, "relative/dir"); err == nil {
		t.Fatal("a relative workspace was accepted")
	}
	if _, err := route(t, workspace(t, "version: 1\ngenerators:\n  - type: crayon\n    output: docs/x\n")); err == nil {
		t.Fatal("invalid settings were accepted")
	}
}
