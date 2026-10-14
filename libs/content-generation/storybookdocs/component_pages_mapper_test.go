package storybookdocs

import (
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func rendered(title, name string, meta storyMeta) renderedStory {
	return renderedStory{Story: Story{ID: slug(title) + "--" + slug(name), Name: name, Title: title, ImportPath: "./x.stories.tsx"}, PNG: []byte("png-" + name), Meta: meta}
}

func fileBody(t *testing.T, files []generatedfile.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return string(file.Body)
		}
	}
	t.Fatalf("no file %s in %v", path, paths(files))

	return ""
}

func paths(files []generatedfile.File) []string {
	var out []string
	for _, file := range files {
		out = append(out, file.Path)
	}

	return out
}

func TestComponentPagesWritesAPagePerComponentAndAnIndex(t *testing.T) {
	meta := storyMeta{
		ArgTypes: map[string]map[string]any{
			"size":   {"description": "How big", "table": map[string]any{"type": map[string]any{"summary": "sm | md"}, "defaultValue": map[string]any{"summary": "md"}}},
			"hidden": {"table": map[string]any{"disable": true}},
		},
		Docs: map[string]any{
			"description": map[string]any{"component": "A button."},
			"source":      map[string]any{"originalSource": "<Button>Go</Button>"},
		},
	}
	files, warnings := ComponentPages([]renderedStory{
		rendered("Components/Button", "Primary", meta),
		rendered("Components/Button", "Secondary", storyMeta{}),
	}, "Library", "storybook-static")
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}

	page := fileBody(t, files, "components/button/README.md")
	for _, want := range []string{
		"# Components / Button", "A button.", "| `size` | sm \\| md | md | How big |",
		"### Primary", "![Primary](./images/primary.png)", "<Button>Go</Button>", "### Secondary",
		"Story source: `./x.stories.tsx`",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("page lacks %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "hidden") {
		t.Fatalf("a disabled prop is listed:\n%s", page)
	}

	parent := fileBody(t, files, "components/README.md")
	if !strings.Contains(parent, "- [Button](./button/README.md)") {
		t.Fatalf("the folder page does not list the component:\n%s", parent)
	}
	index := fileBody(t, files, "README.md")
	if !strings.Contains(index, "# Library") || !strings.Contains(index, "- [Components](./components/README.md)") || !strings.Contains(index, "`storybook-static`") {
		t.Fatalf("index = %s", index)
	}

	var assets int
	for _, file := range files {
		if file.Asset {
			assets++
			if !strings.HasPrefix(file.Path, "components/button/images/") || !strings.HasSuffix(file.Path, ".png") {
				t.Fatalf("asset at %s", file.Path)
			}
		}
	}
	if assets != 2 {
		t.Fatalf("assets = %d, want 2 (%v)", assets, paths(files))
	}
}

func TestComponentPagesKeepsFailuresOutOfThePages(t *testing.T) {
	broken := renderedStory{Story: Story{ID: "a--b", Name: "Broken", Title: "A"}, Failure: "the story threw while rendering"}
	noisy := rendered("A", "Noisy", storyMeta{})
	noisy.Errors = []string{"boom", "again"}
	files, warnings := ComponentPages([]renderedStory{broken, noisy}, "Library", "https://example.test/")

	page := fileBody(t, files, "a/README.md")
	if strings.Contains(page, "threw") || strings.Contains(page, "boom") {
		t.Fatalf("a failure detail is in the page, which would change it run to run:\n%s", page)
	}
	if !strings.Contains(page, "could not be rendered") {
		t.Fatalf("page does not say the example is missing:\n%s", page)
	}
	if len(warnings) != 2 || !strings.Contains(warnings[0], "the story threw") || !strings.Contains(warnings[1], "2 console error(s)") {
		t.Fatalf("warnings = %v", warnings)
	}
	if !strings.Contains(fileBody(t, files, "README.md"), "[https://example.test/](https://example.test/)") {
		t.Fatal("the index does not link to the Storybook URL")
	}
}

func TestComponentPagesGivesCollidingNamesTheirOwnFiles(t *testing.T) {
	files, _ := ComponentPages([]renderedStory{
		rendered("A/B C", "One", storyMeta{}),
		rendered("A/B-C", "Two", storyMeta{}),
		rendered("A/B-C", "one!", storyMeta{}),
	}, "Library", "x")

	names := paths(files)
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			t.Fatalf("path %s is written twice: %v", name, names)
		}
		seen[name] = true
	}
}
