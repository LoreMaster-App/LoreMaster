package sitedeployment

import (
	"strings"
	"testing"
)

func TestCuppaGetsABuildStepIntoTheUploadedFolder(t *testing.T) {
	plan := Recommend(DetectDeployment(map[string]string{"pages.yml": cuppaPages}), nil)

	if plan.Approach != "build" || plan.OutDir != "apps/cuppa-web/frontend/dist/docs" {
		t.Fatalf("unexpected: %+v", plan)
	}
	if !strings.Contains(plan.WorkflowSnippet, "out: apps/cuppa-web/frontend/dist/docs") {
		t.Errorf("snippet: %q", plan.WorkflowSnippet)
	}
	joined := strings.Join(plan.Warnings, "\n")
	for _, want := range []string{"paths:", "no github-pages output"} {
		if !strings.Contains(strings.ToLower(joined), strings.ToLower(want)) {
			t.Errorf("warnings miss %q:\n%s", want, joined)
		}
	}
}

func TestABranchDeployGetsAPathAndAWarningWhenItWouldDeleteTheDocs(t *testing.T) {
	deploy := Deployment{Kind: KindBranchPush, Branch: "gh-pages"}

	plan := Recommend(deploy, []Output{{Branch: "gh-pages"}})

	if plan.Approach != "publish" || plan.Path != "docs" || !strings.Contains(plan.OutputSnippet, "path: docs") {
		t.Fatalf("unexpected: %+v", plan)
	}
	joined := strings.Join(plan.Warnings, "\n")
	if !strings.Contains(joined, "keep_files") || !strings.Contains(joined, "would be refused") {
		t.Errorf("warnings:\n%s", joined)
	}
}

func TestNoDeployLetsLoreMasterOwnTheBranch(t *testing.T) {
	plan := Recommend(Deployment{Kind: KindNone}, []Output{{}})

	if plan.Approach != "publish-root" || plan.Path != "" || plan.OutDir != "" {
		t.Fatalf("unexpected: %+v", plan)
	}
}

func TestSeveralOutputsAskForAnOutputPosition(t *testing.T) {
	plan := Recommend(Deployment{Kind: KindNone}, []Output{{}, {Path: "api"}})

	if !strings.Contains(strings.Join(plan.Warnings, "\n"), "2 github-pages outputs") {
		t.Errorf("warnings: %v", plan.Warnings)
	}
}
