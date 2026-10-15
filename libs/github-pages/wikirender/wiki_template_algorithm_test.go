package wikirender

import (
	"strings"
	"testing"
)

func TestAWikiTemplateWrapsEveryPageButNotTheSidebar(t *testing.T) {
	shell := "> Generated from `{{source}}`\n\n# {{title}}\n\n{{content}}\n\n---\nInternal docs\n"

	got := contents(GenerateWikiTemplated(tree(t, map[string]string{
		"README.md":    "# Project\n\nHello.\n",
		"docs/free.md": "# Free\n\nText.\n",
	}), shell))

	home := got["Home.md"]
	for _, want := range []string{"> Generated from `README.md`", "# Project\n", "Hello.", "Internal docs"} {
		if !strings.Contains(home, want) {
			t.Errorf("Home.md lacks %q:\n%s", want, home)
		}
	}
	if strings.Contains(got["_Sidebar.md"], "Internal docs") {
		t.Errorf("the sidebar was wrapped:\n%s", got["_Sidebar.md"])
	}
	if !strings.Contains(got["Free.md"], "Generated from `docs/free.md`") {
		t.Errorf("Free.md:\n%s", got["Free.md"])
	}
}

func TestWithoutATemplateThePagesAreUnchanged(t *testing.T) {
	files := map[string]string{"README.md": "# Project\n\nHello.\n"}

	if contents(GenerateWiki(tree(t, files)))["Home.md"] != contents(GenerateWikiTemplated(tree(t, files), ""))["Home.md"] {
		t.Fatal("an empty template changed the page")
	}
}

func TestWikiShellProblems(t *testing.T) {
	if got := ShellProblems("# {{title}}"); len(got) != 1 || !strings.Contains(got[0], "{{content}}") {
		t.Fatalf("no content: %v", got)
	}
	if got := ShellProblems("{{content}} {{updated}}"); len(got) != 1 || !strings.Contains(got[0], "{{updated}}") {
		t.Fatalf("unknown: %v", got)
	}
	if got := ShellProblems("{{ content }}"); len(got) != 0 {
		t.Fatalf("valid refused: %v", got)
	}
}
