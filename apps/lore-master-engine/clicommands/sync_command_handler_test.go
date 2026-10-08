package clicommands

import (
	"errors"
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

var cloudCredentials = map[string]string{envEmail: "me@acme.com", envToken: "secret"}

// syncEngine is a double for the whole Confluence sync, with the plan and the execute result
// it should answer.
func syncEngine(plan rpcprotocol.SyncPlanResult, result rpcprotocol.SyncExecuteResult, outputs ...rpcprotocol.Output) *engineDouble {
	if len(outputs) == 0 {
		outputs = []rpcprotocol.Output{confluenceOutput}
	}

	return newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: settingsAnswer(outputs...),
		rpcprotocol.MethodSessionOpen: func(rpcserver.Call) (any, error) {
			return rpcprotocol.SessionOpenResult{SessionID: "s1", BaseURL: "https://acme.atlassian.net/wiki"}, nil
		},
		rpcprotocol.MethodSessionClose:  func(rpcserver.Call) (any, error) { return nil, nil },
		rpcprotocol.MethodSyncPlan:      func(rpcserver.Call) (any, error) { return plan, nil },
		rpcprotocol.MethodSyncExecute:   func(rpcserver.Call) (any, error) { return result, nil },
		rpcprotocol.MethodGeneratorsRun: func(rpcserver.Call) (any, error) { return rpcprotocol.GeneratorsRunResult{}, nil },
		rpcprotocol.MethodPagesPublish: func(rpcserver.Call) (any, error) {
			return rpcprotocol.PagesPublishResult{Changed: true, Files: 3, Branch: "gh-pages", Commit: "abc1234"}, nil
		},
	})
}

var changes = rpcprotocol.SyncPlanResult{
	PlanID: "plan-1",
	Counts: map[string]int{"create": 1, "update": 1, "unchanged": 5},
	Actions: []rpcprotocol.PlanAction{
		{Kind: "create", Path: "docs/new.md", Title: "ENG: New"},
		{Kind: "update", Path: "README.md", Title: "ENG: Home"},
		{Kind: "unchanged", Path: "docs/same.md"},
	},
}

var written = rpcprotocol.SyncExecuteResult{Pages: []rpcprotocol.PageOutcome{{Path: "docs/new.md", Outcome: "written"}, {Path: "README.md", Outcome: "written"}}}

func TestSyncWithoutYesOnlyShowsThePlan(t *testing.T) {
	double := syncEngine(changes, written)

	stdout, _, code := double.run(t, cloudCredentials, "sync")

	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"output 0: Confluence ENG", "plan: create 1, unchanged 5, update 1", "  create       docs/new.md", "  update       README.md", "(pass --yes to apply)"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("missing %q in\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "docs/same.md") {
		t.Fatalf("an unchanged page is listed:\n%s", stdout)
	}
	if double.called(rpcprotocol.MethodSyncExecute) {
		t.Fatal("the plan was executed without --yes")
	}
	if !double.called(rpcprotocol.MethodSessionClose) {
		t.Fatal("the session was left open")
	}
}

func TestSyncWithYesAppliesThePlanAndReportsWhatWasWritten(t *testing.T) {
	double := syncEngine(changes, written)

	stdout, _, code := double.run(t, cloudCredentials, "sync", "--yes")

	if code != ExitOK || !strings.Contains(stdout, "  written 2") {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var params rpcprotocol.SyncExecuteParams
	double.paramsOf(t, rpcprotocol.MethodSyncExecute, &params)
	if params.PlanID != "plan-1" || params.Force || params.Prune {
		t.Fatalf("execute params %+v", params)
	}
	var open rpcprotocol.SessionOpenParams
	double.paramsOf(t, rpcprotocol.MethodSessionOpen, &open)
	if open.BaseURL != "https://acme.atlassian.net/wiki" || open.Credential.Kind != "apitoken" || open.Credential.Email != "me@acme.com" {
		t.Fatalf("session params %+v", open)
	}
}

func TestSyncDryRunChangesNothingEvenWithYes(t *testing.T) {
	double := syncEngine(changes, written)

	stdout, _, code := double.run(t, cloudCredentials, "sync", "--yes", "--dry-run")

	if code != ExitOK || double.called(rpcprotocol.MethodSyncExecute) || !strings.Contains(stdout, "dry run: nothing was changed") {
		t.Fatalf("exit %d: %s", code, stdout)
	}
}

func TestSyncStopsWithExitTwoWhenPagesWereEditedOnThePlatform(t *testing.T) {
	plan := rpcprotocol.SyncPlanResult{PlanID: "p", Counts: map[string]int{"conflict": 2, "update": 1}, Actions: []rpcprotocol.PlanAction{{Kind: "conflict", Path: "a.md", Reason: "edited on the platform"}}}
	double := syncEngine(plan, written)

	stdout, stderr, code := double.run(t, cloudCredentials, "sync", "--yes")

	if code != ExitBlocked || double.called(rpcprotocol.MethodSyncExecute) {
		t.Fatalf("exit %d, executed %v", code, double.called(rpcprotocol.MethodSyncExecute))
	}
	if !strings.Contains(stderr, "2 page(s) were edited on the platform since the last sync; pass --force") || !strings.Contains(stdout, "conflict") {
		t.Fatalf("stdout %q stderr %q", stdout, stderr)
	}
}

func TestSyncForceOverwritesAndPruneTrashesOrphans(t *testing.T) {
	plan := rpcprotocol.SyncPlanResult{PlanID: "p", Counts: map[string]int{"conflict": 1}}
	double := syncEngine(plan, written)

	_, _, code := double.run(t, cloudCredentials, "sync", "--yes", "--force", "--prune")

	var params rpcprotocol.SyncExecuteParams
	double.paramsOf(t, rpcprotocol.MethodSyncExecute, &params)
	if code != ExitOK || !params.Force || !params.Prune {
		t.Fatalf("exit %d, params %+v", code, params)
	}
}

func TestSyncStopsWithExitTwoWhenThePlanHasErrors(t *testing.T) {
	plan := rpcprotocol.SyncPlanResult{PlanID: "p", Errors: []string{"two pages would be titled \"ENG: Guide\""}, Counts: map[string]int{"create": 2}}
	double := syncEngine(plan, written)

	_, stderr, code := double.run(t, cloudCredentials, "sync", "--yes")

	if code != ExitBlocked || double.called(rpcprotocol.MethodSyncExecute) || !strings.Contains(stderr, `error: two pages would be titled "ENG: Guide"`) {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestSyncFailsWhenAPageCouldNotBeWritten(t *testing.T) {
	result := rpcprotocol.SyncExecuteResult{Pages: []rpcprotocol.PageOutcome{{Path: "a.md", Outcome: "written"}, {Path: "b.md", Outcome: "failed", Error: "title already exists"}}}

	stdout, _, code := syncEngine(changes, result).run(t, cloudCredentials, "sync", "--yes")

	if code != ExitFailed || !strings.Contains(stdout, "failed: b.md: title already exists") || !strings.Contains(stdout, "failed 1, written 1") {
		t.Fatalf("exit %d: %s", code, stdout)
	}
}

func TestSyncNamesTheVariablesWhenThereIsNoCredential(t *testing.T) {
	double := syncEngine(changes, written)

	_, stderr, code := double.run(t, nil, "sync", "--yes")

	if code != ExitFailed || !strings.Contains(stderr, "LORE_MASTER_TOKEN") || double.called(rpcprotocol.MethodSessionOpen) {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestSyncReportsARefusedCredentialAndNeverPrintsIt(t *testing.T) {
	double := syncEngine(changes, written)
	double.answer(rpcprotocol.MethodSessionOpen, func(rpcserver.Call) (any, error) { return nil, errors.New("the credential was refused") })

	stdout, stderr, code := double.run(t, cloudCredentials, "sync", "--yes")

	if code != ExitFailed || !strings.Contains(stderr, "signing in to https://acme.atlassian.net/wiki: the credential was refused") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Contains(stdout+stderr, "secret") {
		t.Fatalf("the token leaked:\n%s%s", stdout, stderr)
	}
}

func TestSyncScopesTheCallAndSelectsOutputs(t *testing.T) {
	double := syncEngine(changes, written, confluenceOutput, confluenceOutput)

	_, _, code := double.run(t, cloudCredentials, "sync", "--output", "1", "--scope", "docs/a.md", "--scope", "docs/b.md")

	var plan rpcprotocol.SyncPlanParams
	double.paramsOf(t, rpcprotocol.MethodSyncPlan, &plan)
	if code != ExitOK || plan.Output != 1 || plan.SessionID != "s1" || len(plan.Scope) != 2 || plan.Scope[1] != "docs/b.md" {
		t.Fatalf("exit %d, params %+v", code, plan)
	}
}

func TestSyncPublishesAGitHubPagesOutputOnlyWithYes(t *testing.T) {
	double := syncEngine(changes, written, pagesOutput)

	stdout, _, code := double.run(t, nil, "sync")
	if code != ExitOK || double.called(rpcprotocol.MethodPagesPublish) || !strings.Contains(stdout, "would publish the site") {
		t.Fatalf("without --yes: exit %d: %s", code, stdout)
	}

	stdout, _, code = double.run(t, nil, "sync", "--yes")
	if code != ExitOK || !double.called(rpcprotocol.MethodPagesPublish) || !strings.Contains(stdout, "published 3 files to gh-pages (abc1234)") {
		t.Fatalf("with --yes: exit %d: %s", code, stdout)
	}
	if double.called(rpcprotocol.MethodSessionOpen) {
		t.Fatal("GitHub Pages needs no session")
	}
}

func TestSyncFailsWhenThePublishHasErrors(t *testing.T) {
	double := syncEngine(changes, written, pagesOutput)
	double.answer(rpcprotocol.MethodPagesPublish, func(rpcserver.Call) (any, error) {
		return rpcprotocol.PagesPublishResult{Errors: []string{"push was rejected"}}, nil
	})

	_, stderr, code := double.run(t, nil, "sync", "--yes")

	if code != ExitFailed || !strings.Contains(stderr, "error: push was rejected") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestSyncGenerateRunsTheGeneratorsFirstAndStopsWhenOneFails(t *testing.T) {
	double := syncEngine(changes, written)

	stdout, _, code := double.run(t, cloudCredentials, "sync", "--generate")
	if code != ExitOK || double.calls[0] != rpcprotocol.MethodGeneratorsRun || !strings.Contains(stdout, "plan:") {
		t.Fatalf("exit %d, calls %v", code, double.calls)
	}

	failing := syncEngine(changes, written)
	failing.answer(rpcprotocol.MethodGeneratorsRun, func(rpcserver.Call) (any, error) {
		return rpcprotocol.GeneratorsRunResult{Runs: []rpcprotocol.GeneratorRun{{Type: "ts-docs", Output: "docs/ts", Error: "TypeDoc was not found"}}}, nil
	})
	_, stderr, code := failing.run(t, cloudCredentials, "sync", "--generate", "--yes")
	if code != ExitFailed || failing.called(rpcprotocol.MethodSyncPlan) || !strings.Contains(stderr, "a generator failed") {
		t.Fatalf("exit %d: %s (calls %v)", code, stderr, failing.calls)
	}
}

func TestSyncWithNoConfiguredOutputIsAFailure(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){rpcprotocol.MethodSettingsRead: settingsAnswer(scaffoldOutput)})

	_, stderr, code := double.run(t, cloudCredentials, "sync", "--yes")

	if code != ExitFailed || !strings.Contains(stderr, "there is no configured output to sync") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestSyncTheWorstOutputDecidesTheExitCode(t *testing.T) {
	double := syncEngine(changes, written, confluenceOutput, pagesOutput)
	double.answer(rpcprotocol.MethodPagesPublish, func(rpcserver.Call) (any, error) {
		return rpcprotocol.PagesPublishResult{Errors: []string{"push was rejected"}}, nil
	})

	_, _, code := double.run(t, cloudCredentials, "sync", "--yes")

	if code != ExitFailed || !double.called(rpcprotocol.MethodSyncExecute) {
		t.Fatalf("exit %d: the first output should still have been synced", code)
	}
}

func TestSyncPrintsOneJSONDocument(t *testing.T) {
	double := syncEngine(changes, written, confluenceOutput, pagesOutput)

	stdout, _, code := double.run(t, cloudCredentials, "sync", "--yes", "--json", "--generate")

	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{`"outputs": [`, `"platform": "confluence"`, `"plan": {`, `"result": {`, `"platform": "github-pages"`, `"pages": {`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("missing %s in\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "output 0:") || strings.Contains(stdout, "plan:") {
		t.Fatalf("text leaked into the JSON:\n%s", stdout)
	}
}
