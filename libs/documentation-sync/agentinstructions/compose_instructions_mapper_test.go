package agentinstructions

import (
	"os"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documenttree"
)

func confluence(prefix string, direction string, mermaid string, roots []string, excludes []string) workspacesettings.Output {
	return workspacesettings.Output{
		Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG", TitlePrefix: prefix,
		Direction: direction, MermaidMode: mermaid,
		Content: []workspacesettings.Content{{Type: "markdown", Roots: roots, Excludes: excludes}},
	}
}

func golden(t *testing.T, name string, got string) {
	t.Helper()
	path := "testdata/" + name + ".golden.md"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if normalised := strings.ReplaceAll(string(want), "\r\n", "\n"); got != normalised {
		t.Fatalf("instructions differ from %s (run with UPDATE_GOLDEN=1 to refresh it)\n got:\n%s\nwant:\n%s", path, got, normalised)
	}
}

func TestComposeForAWorkspaceWithoutAConfigSaysSo(t *testing.T) {
	golden(t, "no-config", Compose(workspacesettings.Settings{}, false, documenttree.NestingConventions()))
}

func TestComposeDescribesAConfluenceWorkspaceWithGenerators(t *testing.T) {
	settings := workspacesettings.Settings{
		Version: 1,
		Ignore:  []string{"drafts/**", "NOTES.md"},
		Generators: []workspacesettings.Generator{
			{Type: "test-results", Output: "docs/tests"},
			{Type: "go-docs", Output: "docs/api/"},
		},
		Outputs: []workspacesettings.Output{confluence("ENG", "two-way", "image", []string{"docs", "guides"}, []string{"internal/"})},
	}

	golden(t, "confluence-two-way", Compose(settings, true, documenttree.NestingConventions()))
}

func TestComposeDescribesSeveralStoragesAndTheirSettings(t *testing.T) {
	notIgnored := false
	settings := workspacesettings.Settings{
		Version:        1,
		SkipGitignored: &notIgnored,
		Outputs: []workspacesettings.Output{
			confluence("", "to-platform", "code", []string{"."}, nil),
			{Platform: "github-pages", Content: []workspacesettings.Content{{Type: "markdown", Roots: []string{"docs"}}}},
		},
	}

	golden(t, "two-storages", Compose(settings, true, documenttree.NestingConventions()))
}

func TestComposeQuotesTheLiveRulesInOrder(t *testing.T) {
	conventions := documenttree.NestingConventions()

	text := Compose(workspacesettings.Settings{}, false, conventions)

	last := -1
	for _, convention := range conventions {
		at := strings.Index(text, convention.Summary)
		if at < 0 {
			t.Fatalf("the rule %q is not quoted", convention.Key)
		}
		if at < last {
			t.Fatalf("the rule %q is out of order", convention.Key)
		}
		last = at
	}
}

func TestComposeTellsTheAgentWhatItMayWriteInTheAnnotation(t *testing.T) {
	text := Compose(workspacesettings.Settings{}, false, documenttree.NestingConventions())

	for _, want := range []string{"you may write only `parent:` and `title:`", "YAML front matter does not set a parent or a title", "`place_document`", "`validate_document`"} {
		if !strings.Contains(text, want) {
			t.Errorf("the instructions lack %q", want)
		}
	}
}

func TestComposeIsDeterministic(t *testing.T) {
	settings := workspacesettings.Settings{Version: 1, Outputs: []workspacesettings.Output{confluence("ENG", "to-platform", "image", []string{"docs"}, nil)}}

	if Compose(settings, true, documenttree.NestingConventions()) != Compose(settings, true, documenttree.NestingConventions()) {
		t.Fatal("the same input gave different instructions")
	}
}
