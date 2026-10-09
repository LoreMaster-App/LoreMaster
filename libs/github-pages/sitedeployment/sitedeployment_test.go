package sitedeployment

import (
	"strings"
	"testing"
)

const cuppaPages = `name: Pages
on:
  push:
    branches: [main]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - name: Build the page
        run: npm run build
      - uses: actions/upload-pages-artifact@v4
        with:
          path: apps/cuppa-web/frontend/dist
  deploy:
    needs: build
    steps:
      - id: deployment
        uses: actions/deploy-pages@v5
`

func TestDetectsAnActionsSourceDeployAndItsUploadedFolder(t *testing.T) {
	got := DetectDeployment(map[string]string{".github/workflows/pages.yml": cuppaPages, ".github/workflows/ci.yml": "name: CI\n"})

	if got.Kind != KindActionsSource || got.ArtifactPath != "apps/cuppa-web/frontend/dist" {
		t.Fatalf("unexpected: %+v", got)
	}
	if len(got.Workflows) != 1 || got.Workflows[0] != ".github/workflows/pages.yml" {
		t.Errorf("workflows: %v", got.Workflows)
	}
}

func TestDetectsABranchPushAndWhetherItKeepsOtherFiles(t *testing.T) {
	keeps := `jobs:
  deploy:
    steps:
      - uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./dist
          publish_branch: live
          keep_files: true
`
	got := DetectDeployment(map[string]string{"w.yml": keeps})
	if got.Kind != KindBranchPush || got.Branch != "live" || !got.KeepsOtherFiles {
		t.Fatalf("unexpected: %+v", got)
	}

	cleans := strings.Replace(keeps, "keep_files: true", "force_orphan: true", 1)
	got = DetectDeployment(map[string]string{"w.yml": cleans})
	if got.KeepsOtherFiles {
		t.Errorf("a deploy that does not say keep_files must not be reported as keeping files: %+v", got)
	}
}

func TestDetectsNothingWhenNoWorkflowDeploysToPages(t *testing.T) {
	got := DetectDeployment(map[string]string{"ci.yml": "jobs:\n  t:\n    steps:\n      - run: npm test\n"})

	if got.Kind != KindNone || len(got.Workflows) != 0 {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestAnActionsDeployWinsOverABranchPushAndSaysSo(t *testing.T) {
	got := DetectDeployment(map[string]string{
		"a.yml": cuppaPages,
		"b.yml": "steps:\n  - uses: JamesIves/github-pages-deploy-action@v4\n    with:\n      branch: gh-pages\n",
	})

	if got.Kind != KindActionsSource || len(got.Notes) == 0 {
		t.Fatalf("unexpected: %+v", got)
	}
}

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
