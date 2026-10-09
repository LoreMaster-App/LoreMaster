package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"lore-master/libs/github-pages/sitedeployment"
)

const cuppaWorkflow = `name: Pages
jobs:
  build:
    steps:
      - uses: actions/upload-pages-artifact@v4
        with:
          path: apps/cuppa-web/frontend/dist
  deploy:
    steps:
      - uses: actions/deploy-pages@v5
`

const pagesSettings = "version: 1\noutputs:\n  - platform: github-pages\n    direction: to-platform\n    content:\n      - type: markdown\n        roots: [\".\"]\n        template: default\n"

func text(result toolCallResult) string {
	if len(result.Content) == 0 {
		return ""
	}

	return result.Content[0].Text
}

func TestSitePublishingPlanRecommendsBuildingIntoTheAppsArtifact(t *testing.T) {
	root := workspace(t, map[string]string{
		".github/workflows/pages.yml": cuppaWorkflow,
		"README.md":                   "# Home\n",
	})

	result := toolCall(t, root, sitePublishingPlanToolName, "{}")

	if result.IsError {
		t.Fatalf("error result: %s", text(result))
	}
	for _, want := range []string{"deploys GitHub Pages from an artifact", "apps/cuppa-web/frontend/dist/docs", "actions/pages@main", "no github-pages output"} {
		if !strings.Contains(strings.ToLower(text(result)), strings.ToLower(want)) {
			t.Errorf("missing %q in\n%s", want, text(result))
		}
	}
	raw, _ := json.Marshal(result.StructuredContent)
	var plan sitedeployment.Plan
	if err := json.Unmarshal(raw, &plan); err != nil || plan.Approach != "build" || plan.OutDir != "apps/cuppa-web/frontend/dist/docs" {
		t.Fatalf("structured plan: %+v (%v)", plan, err)
	}
}

func TestSitePublishingPlanFindsTheWorkflowsOfTheRepositoryAboveTheWorkspace(t *testing.T) {
	root := workspace(t, map[string]string{
		".git/HEAD":                   "ref: refs/heads/main\n",
		".github/workflows/pages.yml": cuppaWorkflow,
		"site/README.md":              "# Home\n",
	})

	result := toolCall(t, root+"/site", sitePublishingPlanToolName, "{}")

	if !strings.Contains(text(result), "uploading `apps/cuppa-web/frontend/dist`") {
		t.Fatalf("the repository's workflow was not found:\n%s", text(result))
	}
}

func TestSitePublishingPlanSaysSoWhenNothingDeploysPages(t *testing.T) {
	root := workspace(t, map[string]string{".git/HEAD": "x", "README.md": "# Home\n"})

	result := toolCall(t, root, sitePublishingPlanToolName, "{}")

	if !strings.Contains(text(result), "No .github/workflows folder was found") || !strings.Contains(text(result), "PUBLISH the site to the branch") {
		t.Fatalf("unexpected:\n%s", text(result))
	}
}

func TestPreviewSiteCountsThePagesItWouldBuild(t *testing.T) {
	root := workspace(t, map[string]string{
		".lore-master.yaml": pagesSettings,
		"README.md":         "# Home\n\nWelcome.\n",
		"guide.md":          "# Guide\n",
	})

	result := toolCall(t, root, previewSiteToolName, "{}")

	if result.IsError || !strings.Contains(text(result), "The site would build") {
		t.Fatalf("unexpected:\n%s", text(result))
	}
	raw, _ := json.Marshal(result.StructuredContent)
	var view previewSiteView
	if err := json.Unmarshal(raw, &view); err != nil || view.Pages < 2 || view.Files <= view.Pages {
		t.Fatalf("view: %+v (%v)", view, err)
	}
}

func TestPreviewSiteReportsMarkdownProblemsAsAnError(t *testing.T) {
	root := workspace(t, map[string]string{
		".lore-master.yaml": pagesSettings,
		"a.md":              "<!-- lore-master\nparent: b.md\n-->\n# A\n",
		"b.md":              "<!-- lore-master\nparent: a.md\n-->\n# B\n",
	})

	result := toolCall(t, root, previewSiteToolName, "{}")

	if !result.IsError || !strings.Contains(text(result), "would NOT build") {
		t.Fatalf("unexpected:\n%s", text(result))
	}
}

func TestPreviewSiteWithoutAPagesOutputPointsToThePlanTool(t *testing.T) {
	root := workspace(t, map[string]string{"README.md": "# Home\n"})

	result := toolCall(t, root, previewSiteToolName, "{}")

	if !result.IsError || !strings.Contains(text(result), sitePublishingPlanToolName) {
		t.Fatalf("unexpected:\n%s", text(result))
	}
}
