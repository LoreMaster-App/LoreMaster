package clicommands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/watchcommands"
)

const watchedSettings = `version: 1
outputs:
  - platform: confluence
    baseUrl: https://acme.atlassian.net/wiki
    space: ENG
generators:
  - type: go-docs
    input: [libs/]
    output: docs/api
`

func watchedWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for file, content := range map[string]string{".lore-master.yaml": watchedSettings, "README.md": "# Home\n", "docs/guide.md": "# Guide\n", "libs/core/a.go": "package core\n"} {
		full := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func edit(t *testing.T, root string, file string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(file)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// watching runs `watch` against the double with fast timing until stop is closed or, failing
// that, a deadline; edits are made by do once the watch has started.
type watching struct {
	double *engineDouble
	stdout bytes.Buffer
	stderr bytes.Buffer
	code   int
}

func watchWith(t *testing.T, double *engineDouble, root string, extra []string, do func(), done func() bool) *watching {
	t.Helper()
	// The double's methods run on the engine's goroutines, the test reads them from its own.
	var mu sync.Mutex
	for name, method := range double.methods {
		double.methods[name] = func(ctx context.Context, call rpcserver.Call) (any, error) {
			mu.Lock()
			defer mu.Unlock()

			return method(ctx, call)
		}
	}
	double.methods[rpcprotocol.MethodWatchRoute] = watchcommands.RouteChanges()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result := &watching{double: double}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		args := append([]string{"watch", "--workspace", root, "--debounce", "150ms", "--poll", "20ms"}, extra...)
		result.code = Run(ctx, args, Environment{Getenv: func(name string) string { return cloudCredentials[name] }, Stdout: &result.stdout, Stderr: &result.stderr, WorkingDir: root, Version: "test"}, double.methods)
	}()

	time.Sleep(300 * time.Millisecond)
	do()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		mu.Lock()
		reached := done()
		mu.Unlock()
		if reached {
			break
		}
	}
	time.Sleep(400 * time.Millisecond)
	cancel()
	<-finished

	return result
}

func executed(d *engineDouble) bool { return d.called(rpcprotocol.MethodSyncExecute) }

func TestAnEditedPageSyncsAloneAndNoGeneratorRuns(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, written)

	result := watchWith(t, double, root, []string{"--yes"}, func() { edit(t, root, "docs/guide.md", "# Guide\n\nMore.\n") }, func() bool { return executed(double) })

	if result.code != ExitOK {
		t.Fatalf("exit %d: %s", result.code, result.stderr.String())
	}
	if double.called(rpcprotocol.MethodGeneratorsRun) {
		t.Fatal("a generator ran for a Markdown edit")
	}
	var plan rpcprotocol.SyncPlanParams
	double.paramsOf(t, rpcprotocol.MethodSyncPlan, &plan)
	if !slices.Equal(plan.Scope, []string{"docs/guide.md"}) {
		t.Fatalf("scope %v", plan.Scope)
	}
	out := result.stdout.String()
	for _, want := range []string{"watching ", "changed: docs/guide.md", "syncing docs/guide.md", "done (1 file(s))", "stopped"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

func TestAnEditedSourceRegeneratesOnlyItsGeneratorThenSyncsWhatItWrote(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, written)
	double.answer(rpcprotocol.MethodGeneratorsRun, func(rpcserver.Call) (any, error) {
		return rpcprotocol.GeneratorsRunResult{Runs: []rpcprotocol.GeneratorRun{{Index: 0, Type: "go-docs", Output: "docs/api", Written: []string{"docs/api/core/core.md"}, Unchanged: []string{"docs/api/README.md"}}}}, nil
	})

	result := watchWith(t, double, root, []string{"--yes"}, func() { edit(t, root, "libs/core/a.go", "package core\n\n// New.\n") }, func() bool { return executed(double) })

	if result.code != ExitOK {
		t.Fatalf("exit %d: %s", result.code, result.stderr.String())
	}
	var run rpcprotocol.GeneratorsRunParams
	double.paramsOf(t, rpcprotocol.MethodGeneratorsRun, &run)
	if !slices.Equal(run.Generators, []int{0}) {
		t.Fatalf("generators %v", run.Generators)
	}
	var plan rpcprotocol.SyncPlanParams
	double.paramsOf(t, rpcprotocol.MethodSyncPlan, &plan)
	if !slices.Equal(plan.Scope, []string{"docs/api/core/core.md"}) {
		t.Fatalf("scope %v", plan.Scope)
	}
}

func TestQuickEditsToSeveralFilesAreOneScopedSync(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, written)

	watchWith(t, double, root, []string{"--yes"}, func() {
		edit(t, root, "README.md", "# Home\n\nOne.\n")
		time.Sleep(40 * time.Millisecond)
		edit(t, root, "docs/guide.md", "# Guide\n\nTwo.\n")
	}, func() bool { return executed(double) })

	plans := 0
	for _, call := range double.calls {
		if call == rpcprotocol.MethodSyncPlan {
			plans++
		}
	}
	var plan rpcprotocol.SyncPlanParams
	double.paramsOf(t, rpcprotocol.MethodSyncPlan, &plan)
	if plans != 1 || !slices.Equal(plan.Scope, []string{"README.md", "docs/guide.md"}) {
		t.Fatalf("%d plans, scope %v", plans, plan.Scope)
	}
}

func TestChangedSettingsRegenerateAndSyncEverything(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, written)

	watchWith(t, double, root, []string{"--yes"}, func() { edit(t, root, ".lore-master.yaml", watchedSettings+"# changed\n") }, func() bool { return executed(double) })

	var plan rpcprotocol.SyncPlanParams
	double.paramsOf(t, rpcprotocol.MethodSyncPlan, &plan)
	if len(plan.Scope) != 0 || !double.called(rpcprotocol.MethodGeneratorsRun) {
		t.Fatalf("scope %v, generators run %v", plan.Scope, double.called(rpcprotocol.MethodGeneratorsRun))
	}
}

func TestWithoutYesTheBatchIsPlannedButNotApplied(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, written)

	result := watchWith(t, double, root, nil, func() { edit(t, root, "README.md", "# Home\n\nChanged.\n") }, func() bool { return double.called(rpcprotocol.MethodSyncPlan) })

	if executed(double) || !strings.Contains(result.stdout.String(), "pass --yes to apply") || !strings.Contains(result.stdout.String(), "without --yes batches are only planned") {
		t.Fatalf("executed %v:\n%s", executed(double), result.stdout.String())
	}
}

func TestAFileTouchedWithoutAChangeDoesNothing(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, written)

	watchWith(t, double, root, []string{"--yes"}, func() {
		later := time.Now().Add(time.Hour)
		if err := os.Chtimes(filepath.Join(root, "README.md"), later, later); err != nil {
			t.Fatal(err)
		}
	}, func() bool { return false })

	if len(double.calls) != 0 {
		t.Fatalf("the engine was called: %v", double.calls)
	}
}

func TestOurOwnWritesAreNotTakenForTheNextChange(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, written)
	// A sync writes the annotation back into the page it synced.
	double.answer(rpcprotocol.MethodSyncExecute, func(rpcserver.Call) (any, error) {
		edit(t, root, "docs/guide.md", "<!-- lore-master\npageId: 1\n-->\n# Guide\n\nMore.\n")

		return written, nil
	})

	watchWith(t, double, root, []string{"--yes"}, func() { edit(t, root, "docs/guide.md", "# Guide\n\nMore.\n") }, func() bool { return executed(double) })

	plans := 0
	for _, call := range double.calls {
		if call == rpcprotocol.MethodSyncPlan {
			plans++
		}
	}
	if plans != 1 {
		t.Fatalf("%d plans: the annotation write-back was synced again", plans)
	}
}

func TestAFailingSyncIsTriedAgain(t *testing.T) {
	root := watchedWorkspace(t)
	double := syncEngine(changes, rpcprotocol.SyncExecuteResult{Pages: []rpcprotocol.PageOutcome{{Path: "README.md", Outcome: "failed", Error: "503"}}})

	result := watchWith(t, double, root, []string{"--yes"}, func() { edit(t, root, "README.md", "# Home\n\nX.\n") }, func() bool {
		count := 0
		for _, call := range double.calls {
			if call == rpcprotocol.MethodSyncExecute {
				count++
			}
		}

		return count >= 2
	})

	if !strings.Contains(result.stdout.String(), "failed: the sync failed (see above); trying again in") {
		t.Fatalf("stdout:\n%s", result.stdout.String())
	}
}

func TestWatchRefusesJSONAndNonsenseTiming(t *testing.T) {
	double := newEngineDouble(nil)
	for _, args := range [][]string{{"watch", "--json"}, {"watch", "--debounce", "0s"}, {"watch", "--poll", "-1s"}, {"watch", "--debounce", "soon"}} {
		if _, _, code := double.run(t, nil, args...); code != ExitUsage {
			t.Errorf("%v: exit %d", args, code)
		}
	}
}
