package clicommands

import (
	"errors"
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

var (
	confluenceOutput = rpcprotocol.Output{Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG"}
	scaffoldOutput   = rpcprotocol.Output{Platform: "confluence"}
	pagesOutput      = rpcprotocol.Output{Platform: "github-pages"}
)

func settingsAnswer(outputs ...rpcprotocol.Output) func(rpcserver.Call) (any, error) {
	return func(rpcserver.Call) (any, error) {
		return rpcprotocol.SettingsReadResult{Exists: true, Settings: rpcprotocol.Settings{Version: 1, Outputs: outputs}}, nil
	}
}

func treeAnswer(trees map[int]rpcprotocol.WorkspaceTreeResult) func(rpcserver.Call) (any, error) {
	return func(call rpcserver.Call) (any, error) {
		var params rpcprotocol.WorkspaceTreeParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		tree, found := trees[params.Output]
		if !found {
			return nil, errors.New("settings are invalid")
		}

		return tree, nil
	}
}

func TestTreePrintsEachConfiguredOutputsPagesWithTheirStatus(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: settingsAnswer(scaffoldOutput, confluenceOutput, pagesOutput),
		rpcprotocol.MethodWorkspaceTree: treeAnswer(map[int]rpcprotocol.WorkspaceTreeResult{
			1: {Nodes: []rpcprotocol.TreeNode{
				{Path: "README.md", Title: "Home", Status: "synced"},
				{Path: "docs/guide.md", Title: "Guide", Parent: "README.md", Depth: 1, Status: "local-changes"},
			}, Warnings: []string{"docs/x.md: odd"}},
			2: {Nodes: []rpcprotocol.TreeNode{{Path: "README.md", Title: "Home"}}},
		}),
	})

	stdout, _, code := double.run(t, nil, "tree")

	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"output 1: Confluence ENG (https://acme.atlassian.net/wiki)",
		"[synced]",
		"Home (README.md)",
		"  [local-changes] Guide (docs/guide.md)",
		"  warning: docs/x.md: odd",
		"output 2: GitHub Pages",
		"[-]",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("missing %q in\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "output 0") {
		t.Fatalf("the unconfigured scaffold was shown:\n%s", stdout)
	}
}

func TestTreeShowsOnlyTheOutputsAskedFor(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead:  settingsAnswer(confluenceOutput, pagesOutput),
		rpcprotocol.MethodWorkspaceTree: treeAnswer(map[int]rpcprotocol.WorkspaceTreeResult{1: {}}),
	})

	stdout, _, code := double.run(t, nil, "tree", "--output", "1")

	if code != ExitOK || strings.Contains(stdout, "output 0") || !strings.Contains(stdout, "output 1: GitHub Pages") {
		t.Fatalf("exit %d: %s", code, stdout)
	}
}

func TestTreeRefusesAnOutputThatDoesNotExist(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){rpcprotocol.MethodSettingsRead: settingsAnswer(confluenceOutput)})

	_, stderr, code := double.run(t, nil, "tree", "--output", "4")

	if code != ExitFailed || !strings.Contains(stderr, "output 4 does not exist; the settings have 1") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestTreeFailsWhenAnOutputHasProblemsOrCannotBeRead(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: settingsAnswer(confluenceOutput, pagesOutput),
		rpcprotocol.MethodWorkspaceTree: treeAnswer(map[int]rpcprotocol.WorkspaceTreeResult{
			0: {Problems: []string{"bad.md: could not be read"}},
		}),
	})

	stdout, stderr, code := double.run(t, nil, "tree")

	if code != ExitFailed || !strings.Contains(stdout, "problem: bad.md: could not be read") || !strings.Contains(stderr, "output 1: settings are invalid") {
		t.Fatalf("exit %d: %q %q", code, stdout, stderr)
	}
}

func TestTreePrintsJSONKeyedByOutput(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead:  settingsAnswer(confluenceOutput),
		rpcprotocol.MethodWorkspaceTree: treeAnswer(map[int]rpcprotocol.WorkspaceTreeResult{0: {Nodes: []rpcprotocol.TreeNode{{Path: "README.md", Title: "Home", Status: "new"}}}}),
	})

	stdout, _, code := double.run(t, nil, "tree", "--json")

	if code != ExitOK || !strings.Contains(stdout, `"outputs"`) || !strings.Contains(stdout, `"0": {`) || !strings.Contains(stdout, `"status": "new"`) {
		t.Fatalf("exit %d: %s", code, stdout)
	}
}

func TestTreeListsTheFilesLeftOutWithTheirRule(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: settingsAnswer(pagesOutput),
		rpcprotocol.MethodWorkspaceTree: treeAnswer(map[int]rpcprotocol.WorkspaceTreeResult{0: {
			Nodes: []rpcprotocol.TreeNode{{Path: "README.md", Title: "Home"}},
			LeftOut: []rpcprotocol.LeftOutFile{
				{Path: "CLAUDE.md", Rule: "ignore", Pattern: "CLAUDE.md"},
				{Path: "build/x.md", Rule: "gitignore", Pattern: "build/", Source: ".gitignore"},
				{Path: "notes/y.md", Rule: "outside-roots"},
			},
			LeftOutTotal: 5,
		}}),
	})

	stdout, _, code := double.run(t, nil, "tree")

	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"left out (5):",
		"CLAUDE.md  - ignore (CLAUDE.md)",
		"build/x.md  - gitignore (.gitignore: build/)",
		"notes/y.md  - outside the roots",
		"... and 2 more",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("missing %q in\n%s", want, stdout)
		}
	}
}
