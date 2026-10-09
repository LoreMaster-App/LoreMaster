package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
)

const pagesSettings = "version: 1\noutputs:\n  - platform: github-pages\n    direction: to-platform\n    content:\n      - type: markdown\n        roots: [\".\"]\n        template: default\n"

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
