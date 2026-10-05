package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
)

func placementViewOf(t *testing.T, result toolCallResult) placementView {
	t.Helper()
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	raw, _ := json.Marshal(result.StructuredContent)
	var view placementView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode placement view: %v", err)
	}

	return view
}

func TestPlaceDocumentUsesADottedNameUnderALocalParent(t *testing.T) {
	root := workspace(t, map[string]string{"README.md": "# Home\n"})
	view := placementViewOf(t, toolCall(t, root, placeDocumentToolName, `{"h1":"Architecture Overview","parent":"README.md"}`))
	if view.Path != "readme.architecture-overview.md" {
		t.Errorf("path = %q", view.Path)
	}
	if view.Mechanism != "dotted-name" || view.Parent != "README.md" {
		t.Errorf("mechanism=%q parent=%q", view.Mechanism, view.Parent)
	}
	if view.Annotation != "" {
		t.Errorf("expected no annotation, got %q", view.Annotation)
	}
}

func TestPlaceDocumentResolvesParentByTitle(t *testing.T) {
	root := workspace(t, map[string]string{"docs/guide.md": "# The Guide\n"})
	view := placementViewOf(t, toolCall(t, root, placeDocumentToolName, `{"h1":"Setup","parent":"The Guide"}`))
	if view.Path != "docs/guide.setup.md" || view.Parent != "docs/guide.md" || view.Mechanism != "dotted-name" {
		t.Errorf("view = %+v", view)
	}
}

func TestPlaceDocumentUsesExplicitParentAcrossDirectories(t *testing.T) {
	root := workspace(t, map[string]string{"README.md": "# Home\n"})
	view := placementViewOf(t, toolCall(t, root, placeDocumentToolName, `{"h1":"Deep Note","parent":"README.md","directory":"notes"}`))
	if view.Path != "notes/deep-note.md" {
		t.Errorf("path = %q", view.Path)
	}
	if view.Mechanism != "explicit-parent" || view.Parent != "README.md" {
		t.Errorf("mechanism=%q parent=%q", view.Mechanism, view.Parent)
	}
	if !strings.Contains(view.Annotation, "parent: /README.md") || !strings.Contains(view.Annotation, "lore-master") {
		t.Errorf("annotation missing explicit parent: %q", view.Annotation)
	}
}

func TestPlaceDocumentTopLevelWhenNoParent(t *testing.T) {
	root := workspace(t, map[string]string{"other.md": "# Other\n"})
	view := placementViewOf(t, toolCall(t, root, placeDocumentToolName, `{"h1":"Standalone"}`))
	if view.Path != "standalone.md" || view.Parent != "" || view.Mechanism != "selected-parent" {
		t.Errorf("view = %+v", view)
	}
}

func TestPlaceDocumentAppliesTheTitlePrefix(t *testing.T) {
	root := workspace(t, map[string]string{
		".lore-master.yaml": "version: 1\noutputs:\n  - platform: confluence\n    titlePrefix: ACME\n    content:\n      - type: markdown\n        roots: ['.']\n",
		"README.md":         "# Home\n",
	})
	view := placementViewOf(t, toolCall(t, root, placeDocumentToolName, `{"h1":"Glossary","parent":"README.md"}`))
	if view.PublishedTitle != "ACME: Glossary" {
		t.Errorf("publishedTitle = %q", view.PublishedTitle)
	}
}

func TestPlaceDocumentWarnsOnTitleClash(t *testing.T) {
	root := workspace(t, map[string]string{
		"README.md":  "# Home\n",
		"welcome.md": "# Welcome\n",
	})
	view := placementViewOf(t, toolCall(t, root, placeDocumentToolName, `{"h1":"Welcome","parent":"README.md"}`))
	if !strings.Contains(strings.Join(view.Warnings, "\n"), "already used by welcome.md") {
		t.Errorf("expected a title-clash warning, got %v", view.Warnings)
	}
}

func TestPlaceDocumentReportsUnknownParent(t *testing.T) {
	root := workspace(t, map[string]string{"README.md": "# Home\n"})
	result := toolCall(t, root, placeDocumentToolName, `{"h1":"Thing","parent":"Nope"}`)
	if !result.IsError || !strings.Contains(result.Content[0].Text, "no synced page matches") {
		t.Errorf("expected unknown-parent error, got %+v", result)
	}
}

func TestPlaceDocumentNeedsAnH1(t *testing.T) {
	root := workspace(t, map[string]string{"README.md": "# Home\n"})
	result := toolCall(t, root, placeDocumentToolName, `{}`)
	if !result.IsError || !strings.Contains(result.Content[0].Text, "needs the new page's") {
		t.Errorf("expected missing-h1 error, got %+v", result)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Architecture Overview":  "architecture-overview",
		"  Spaces  &  Symbols! ": "spaces-symbols",
		"CamelCase":              "camelcase",
		"123 Start":              "123-start",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
