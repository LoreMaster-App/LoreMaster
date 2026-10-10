package clicommands

import (
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

func pagesBuildDouble(result rpcprotocol.PagesBuildResult, outputs ...rpcprotocol.Output) *engineDouble {
	return newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: settingsAnswer(outputs...),
		rpcprotocol.MethodPagesBuild:   func(rpcserver.Call) (any, error) { return result, nil },
	})
}

func TestPagesBuildWritesTheSiteIntoTheFolderAndPicksTheOnlyPagesOutput(t *testing.T) {
	double := pagesBuildDouble(rpcprotocol.PagesBuildResult{OutDir: "x", Files: 12}, confluenceOutput, pagesOutput)

	stdout, _, code := double.run(t, nil, "pages", "build", "--out", "dist/docs")

	if code != ExitOK || !strings.Contains(stdout, "built 12 files") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
	var sent rpcprotocol.PagesBuildParams
	double.paramsOf(t, rpcprotocol.MethodPagesBuild, &sent)
	if sent.Output != 1 || !strings.HasSuffix(strings.ReplaceAll(sent.OutDir, "\\", "/"), "dist/docs") {
		t.Fatalf("unexpected params: %+v", sent)
	}
}

func TestPagesBuildNeedsAnOutFolder(t *testing.T) {
	double := pagesBuildDouble(rpcprotocol.PagesBuildResult{}, pagesOutput)

	_, stderr, code := double.run(t, nil, "pages", "build")

	if code != ExitUsage || !strings.Contains(stderr, "--out is required") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
}

func TestPagesBuildStopsWithoutWritingWhenTheMarkdownHasErrors(t *testing.T) {
	double := pagesBuildDouble(rpcprotocol.PagesBuildResult{Errors: []string{"docs/a.md: two H1 headings"}}, pagesOutput)

	stdout, stderr, code := double.run(t, nil, "pages", "build", "--out", "dist/docs")

	if code != ExitBlocked || !strings.Contains(stderr, "two H1 headings") || strings.Contains(stdout, "built") {
		t.Fatalf("exit %d\nstdout %s\nstderr %s", code, stdout, stderr)
	}
}

func TestPagesBuildAsksForAChoiceWhenThereAreSeveralPagesOutputs(t *testing.T) {
	double := pagesBuildDouble(rpcprotocol.PagesBuildResult{}, pagesOutput, pagesOutput)

	_, stderr, code := double.run(t, nil, "pages", "build", "--out", "dist/docs")

	if code != ExitFailed || !strings.Contains(stderr, "choose one with --output") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	if double.called(rpcprotocol.MethodPagesBuild) {
		t.Fatal("a build ran without a choice")
	}
}

func TestPagesBuildFailsWhenThereIsNoPagesOutput(t *testing.T) {
	double := pagesBuildDouble(rpcprotocol.PagesBuildResult{}, confluenceOutput)

	_, stderr, code := double.run(t, nil, "pages", "build", "--out", "dist/docs")

	if code != ExitFailed || !strings.Contains(stderr, "no github-pages output") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
}

func TestPagesPublishReportsTheCommitAndTheSite(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: settingsAnswer(pagesOutput),
		rpcprotocol.MethodPagesPublish: func(rpcserver.Call) (any, error) {
			return rpcprotocol.PagesPublishResult{Branch: "gh-pages", Commit: "abc1234", Changed: true, Files: 7, URL: "https://acme.github.io/site/"}, nil
		},
	})

	stdout, _, code := double.run(t, nil, "pages", "publish")

	if code != ExitOK || !strings.Contains(stdout, "published 7 files to gh-pages (abc1234)") || !strings.Contains(stdout, "https://acme.github.io/site/") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
}

func TestPagesWithoutASubcommandIsAUsageError(t *testing.T) {
	double := pagesBuildDouble(rpcprotocol.PagesBuildResult{}, pagesOutput)

	_, stderr, code := double.run(t, nil, "pages")

	if code != ExitUsage || !strings.Contains(stderr, "pages build") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
}

func pagesCheckDouble(result rpcprotocol.PagesCheckResult) *engineDouble {
	return newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: settingsAnswer(pagesOutput),
		rpcprotocol.MethodPagesCheck:   func(rpcserver.Call) (any, error) { return result, nil },
	})
}

func TestPagesCheckSaysUpToDateAndExitsOK(t *testing.T) {
	double := pagesCheckDouble(rpcprotocol.PagesCheckResult{Branch: "gh-pages", UpToDate: true, Files: 9})

	stdout, _, code := double.run(t, nil, "pages", "check", "--exit-code")

	if code != ExitOK || !strings.Contains(stdout, "gh-pages is up to date (9 files)") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
}

func TestPagesCheckListsTheChangesAndOnlyFailsWithExitCode(t *testing.T) {
	result := rpcprotocol.PagesCheckResult{
		Branch: "gh-pages", ChangesTotal: 3, Files: 9,
		Changes: []rpcprotocol.PagesChange{{Path: "index.html", Kind: "modified"}, {Path: "docs/new.html", Kind: "added"}},
	}

	stdout, _, code := pagesCheckDouble(result).run(t, nil, "pages", "check")
	if code != ExitOK || !strings.Contains(stdout, "3 files would change") || !strings.Contains(stdout, "modified index.html") || !strings.Contains(stdout, "and 1 more") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}

	_, _, code = pagesCheckDouble(result).run(t, nil, "pages", "check", "--exit-code")
	if code != ExitBlocked {
		t.Fatalf("want exit %d with --exit-code, got %d", ExitBlocked, code)
	}
}

func TestPagesCheckStopsWhenTheMarkdownHasErrors(t *testing.T) {
	double := pagesCheckDouble(rpcprotocol.PagesCheckResult{Errors: []string{"docs/a.md: two H1 headings"}})

	_, stderr, code := double.run(t, nil, "pages", "check")

	if code != ExitBlocked || !strings.Contains(stderr, "two H1 headings") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
}
