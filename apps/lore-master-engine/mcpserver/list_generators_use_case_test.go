package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatorregistry"
)

func TestListGeneratorsNamesEveryGeneratorTheEngineHas(t *testing.T) {
	result := toolCall(t, "", listGeneratorsToolName, "{}")

	body := text(result)
	if result.IsError {
		t.Fatalf("the catalogue needs no workspace:\n%s", body)
	}
	for _, kind := range generatorregistry.Types() {
		if !strings.Contains(body, kind) {
			t.Errorf("the tool does not mention the generator %q", kind)
		}
	}
	if !strings.Contains(body, "No workspace is open") {
		t.Errorf("without a workspace it must say the configured ones are unknown:\n%s", body)
	}
}

func TestListGeneratorsShowsWhereTheWorkspaceWritesGeneratedPages(t *testing.T) {
	root := workspace(t, map[string]string{
		settingsFixture: docsSettings + "generators:\n  - type: test-results\n    output: docs/tests\n  - source: storybook-static\n    output: docs/components\n",
	})

	result := toolCall(t, root, listGeneratorsToolName, "{}")

	body := text(result)
	if !strings.Contains(body, "docs/tests") || !strings.Contains(body, "docs/components") {
		t.Fatalf("the output folders are missing:\n%s", body)
	}
	raw, _ := json.Marshal(result.StructuredContent)
	var report generatorsReport
	if err := json.Unmarshal(raw, &report); err != nil || len(report.Configured) != 2 {
		t.Fatalf("configured: %+v (%v)", report.Configured, err)
	}
}
