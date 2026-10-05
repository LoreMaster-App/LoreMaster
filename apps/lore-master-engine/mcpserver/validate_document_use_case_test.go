package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
)

func validationViewOf(t *testing.T, result toolCallResult) validationView {
	t.Helper()
	raw, _ := json.Marshal(result.StructuredContent)
	var view validationView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode validation view: %v", err)
	}

	return view
}

func TestValidateDocumentReportsCleanNesting(t *testing.T) {
	root := workspace(t, map[string]string{
		"README.md":              "# Home\n",
		"readme.architecture.md": "# Architecture\n",
	})
	result := toolCall(t, root, validateDocumentToolName, `{"path":"readme.architecture.md"}`)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	view := validationViewOf(t, result)
	if view.Parent != "README.md" || view.Rule != "dotted-name" || !view.Synced {
		t.Errorf("view = %+v", view)
	}
	if len(view.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", view.Warnings)
	}
}

func TestValidateDocumentFlagsTitleClash(t *testing.T) {
	root := workspace(t, map[string]string{
		"a.md": "# Same Title\n",
		"b.md": "# Same Title\n",
	})
	result := toolCall(t, root, validateDocumentToolName, `{"path":"a.md"}`)
	if !strings.Contains(strings.Join(validationViewOf(t, result).Warnings, "\n"), "also used by b.md") {
		t.Errorf("expected a title-clash warning naming b.md, got: %s", result.Content[0].Text)
	}
}

func TestValidateDocumentFlagsMissingHeading(t *testing.T) {
	root := workspace(t, map[string]string{
		"notes.md": "Just a paragraph, no heading.\n",
	})
	result := toolCall(t, root, validateDocumentToolName, `{"path":"notes.md"}`)
	if !strings.Contains(strings.Join(validationViewOf(t, result).Warnings, "\n"), "no H1 heading") {
		t.Errorf("expected a missing-H1 warning, got: %s", result.Content[0].Text)
	}
}

func TestValidateDocumentFlagsAnExcludedFile(t *testing.T) {
	root := workspace(t, map[string]string{
		".lore-master.yaml": "version: 1\noutputs:\n  - platform: confluence\n    content:\n      - type: markdown\n        roots: [docs]\n",
		"docs/README.md":    "# Docs\n",
		"stray.md":          "# Stray\n",
	})
	result := toolCall(t, root, validateDocumentToolName, `{"path":"stray.md"}`)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	view := validationViewOf(t, result)
	if view.Synced {
		t.Error("expected stray.md to be reported as not synced")
	}
	if !strings.Contains(strings.Join(view.Warnings, "\n"), "will not include it") {
		t.Errorf("expected an excluded-file warning, got: %s", result.Content[0].Text)
	}
}

func TestValidateDocumentRejectsPathOutsideWorkspace(t *testing.T) {
	root := workspace(t, map[string]string{"README.md": "# Home\n"})
	result := toolCall(t, root, validateDocumentToolName, `{"path":"../secret.md"}`)
	if !result.IsError || !strings.Contains(result.Content[0].Text, "outside the workspace") {
		t.Errorf("expected an outside-workspace error, got: %+v", result)
	}
}

func TestValidateDocumentNeedsAPath(t *testing.T) {
	root := workspace(t, map[string]string{"README.md": "# Home\n"})
	result := toolCall(t, root, validateDocumentToolName, `{}`)
	if !result.IsError || !strings.Contains(result.Content[0].Text, "needs a") {
		t.Errorf("expected a missing-path error, got: %+v", result)
	}
}
