package clicommands

import (
	"errors"
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

func generatorsAnswer(runs ...rpcprotocol.GeneratorRun) map[string]func(rpcserver.Call) (any, error) {
	return map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodGeneratorsRun: func(rpcserver.Call) (any, error) { return rpcprotocol.GeneratorsRunResult{Runs: runs}, nil },
	}
}

func TestGeneratePrintsWhatEachGeneratorDid(t *testing.T) {
	double := newEngineDouble(generatorsAnswer(
		rpcprotocol.GeneratorRun{Index: 0, Type: "test-results", Output: "docs/tests", Written: []string{"a.md", "b.md"}, Unchanged: []string{"c.md"}, Warnings: []string{"reports/bad.xml: not valid"}},
		rpcprotocol.GeneratorRun{Index: 1, Type: "go-docs", Output: "docs/api", Removed: []string{"old.md"}},
	))

	stdout, _, code := double.run(t, nil, "generate", "--workspace", t.TempDir())

	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	want := "test-results -> docs/tests: 2 written, 1 unchanged, 0 removed\n  warning: reports/bad.xml: not valid\ngo-docs -> docs/api: 0 written, 0 unchanged, 1 removed\n"
	if stdout != want {
		t.Fatalf("stdout\n%q\nwant\n%q", stdout, want)
	}
}

func TestGenerateRunsTheNamedGeneratorsInTheWorkspaceGiven(t *testing.T) {
	double := newEngineDouble(generatorsAnswer())
	workspace := t.TempDir()

	double.run(t, nil, "generate", "--workspace", workspace, "--generator", "1", "--generator", "3")

	var params rpcprotocol.GeneratorsRunParams
	double.paramsOf(t, rpcprotocol.MethodGeneratorsRun, &params)
	if params.WorkspaceRoot != workspace || len(params.Generators) != 2 || params.Generators[0] != 1 || params.Generators[1] != 3 {
		t.Fatalf("params %+v", params)
	}
}

func TestGenerateUsesTheWorkingFolderByDefault(t *testing.T) {
	double := newEngineDouble(generatorsAnswer())

	double.run(t, nil, "generate")

	var params rpcprotocol.GeneratorsRunParams
	double.paramsOf(t, rpcprotocol.MethodGeneratorsRun, &params)
	if params.WorkspaceRoot == "" || len(params.Generators) != 0 {
		t.Fatalf("params %+v", params)
	}
}

func TestGenerateFailsWhenAGeneratorFailedAndStillPrintsTheOthers(t *testing.T) {
	double := newEngineDouble(generatorsAnswer(
		rpcprotocol.GeneratorRun{Type: "ts-docs", Output: "docs/ts", Error: "TypeDoc was not found; install it with: npm i -D typedoc"},
		rpcprotocol.GeneratorRun{Type: "go-docs", Output: "docs/api", Written: []string{"a.md"}},
	))

	stdout, _, code := double.run(t, nil, "generate")

	if code != ExitFailed {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "ts-docs -> docs/ts: failed\n  error: TypeDoc was not found") || !strings.Contains(stdout, "go-docs -> docs/api: 1 written") {
		t.Fatalf("stdout %q", stdout)
	}
}

func TestGenerateSaysSoWhenThereAreNoGenerators(t *testing.T) {
	stdout, _, code := newEngineDouble(generatorsAnswer()).run(t, nil, "generate")

	if code != ExitOK || stdout != "no generators are configured\n" {
		t.Fatalf("exit %d: %q", code, stdout)
	}
}

func TestGeneratePrintsJSONWhenAsked(t *testing.T) {
	double := newEngineDouble(generatorsAnswer(rpcprotocol.GeneratorRun{Index: 0, Type: "go-docs", Output: "docs/api", Written: []string{"a.md"}}))

	stdout, _, code := double.run(t, nil, "generate", "--json")

	if code != ExitOK || !strings.Contains(stdout, `"generators"`) || !strings.Contains(stdout, `"written": [`) || strings.Contains(stdout, "->") {
		t.Fatalf("exit %d: %s", code, stdout)
	}
}

func TestGenerateReportsAnEngineErrorAsAFailure(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodGeneratorsRun: func(rpcserver.Call) (any, error) { return nil, errors.New("outputs[0].mermaidMode \"x\" is not valid") },
	})

	stdout, stderr, code := double.run(t, nil, "generate")

	if code != ExitFailed || stdout != "" || !strings.Contains(stderr, `error: outputs[0].mermaidMode "x" is not valid`) {
		t.Fatalf("exit %d: %q %q", code, stdout, stderr)
	}
}
