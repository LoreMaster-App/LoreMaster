package mcpserver

import (
	"strings"
	"testing"
)

const docsSettings = "version: 1\noutputs:\n  - platform: github-pages\n    direction: to-platform\n    content:\n      - type: markdown\n        roots: [\".\"]\n        template: default\n"

func TestPlanMovesSaysWhereAMovedPageWillNest(t *testing.T) {
	root := workspace(t, map[string]string{
		settingsFixture:     docsSettings,
		"guide/README.md":       "# Guide\n",
		"misc/setup.md":         "# Setup\n",
		"misc/setup.windows.md": "# Setup on Windows\n",
	})

	result := toolCall(t, root, planMovesToolName, `{"moves":[{"from":"misc/setup.md","to":"guide/setup.md"}]}`)

	body := text(result)
	if result.IsError {
		t.Fatalf("unexpected error:\n%s", body)
	}
	for _, want := range []string{"misc/setup.md", "under guide/README.md", "misc/setup.windows.md -> guide/setup.windows.md"} {
		if !strings.Contains(body, want) {
			t.Errorf("the plan lacks %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, "Nothing was moved") {
		t.Errorf("the plan must say it wrote nothing:\n%s", body)
	}
}

func TestPlanMovesKeepsAnExplicitParentNamingTheSameDocument(t *testing.T) {
	root := workspace(t, map[string]string{
		settingsFixture: docsSettings,
		"a/index.md":        "# Index\n",
		"a/page.md":         "<!-- lore-master\nparent: index.md\n-->\n# Page\n",
	})

	result := toolCall(t, root, planMovesToolName, `{"moves":[{"from":"a/page.md","to":"b/page.md"}]}`)

	if result.IsError || !strings.Contains(text(result), "parent: ../a/index.md") {
		t.Fatalf("the parent line must be rewritten:\n%s", text(result))
	}
}

func TestPlanMovesRefusesPathsOutsideTheWorkspaceAndEmptyInput(t *testing.T) {
	root := workspace(t, map[string]string{settingsFixture: docsSettings, "a.md": "# A\n"})

	for _, arguments := range []string{`{}`, `{"moves":[{"from":"a.md","to":"../outside.md"}]}`} {
		if result := toolCall(t, root, planMovesToolName, arguments); !result.IsError {
			t.Errorf("%s should be refused:\n%s", arguments, text(result))
		}
	}
}
