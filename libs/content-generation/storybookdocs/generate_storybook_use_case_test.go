package storybookdocs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

// fakeRenderer answers every story with a PNG named after it, and counts the calls.
type fakeRenderer struct {
	rendered atomic.Int32
	fail     map[string]bool
}

func (f *fakeRenderer) Render(_ context.Context, base string, story Story) renderedStory {
	f.rendered.Add(1)
	if f.fail[story.ID] {
		return renderedStory{Story: story, Failure: "the story threw while rendering"}
	}

	return renderedStory{Story: story, PNG: []byte("png:" + story.ID + "@" + strings.Split(base, ":")[0])}
}

func (f *fakeRenderer) Close() {}

func writeBuild(t *testing.T, index string) string {
	t.Helper()
	root := t.TempDir()
	folder := filepath.Join(root, "storybook-static")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

const twoComponents = `{"v":5,"entries":{
	"c-button--primary":{"id":"c-button--primary","type":"story","name":"Primary","title":"Components/Button"},
	"c-button--ghost":{"id":"c-button--ghost","type":"story","name":"Ghost","title":"Components/Button"},
	"i-debug--main":{"id":"i-debug--main","type":"story","name":"Main","title":"Internal/Debug"}
}}`

func startFake(fake *fakeRenderer) func(context.Context) (storyRenderer, error) {
	return func(context.Context) (storyRenderer, error) { return fake, nil }
}

func TestGenerateRendersTheBuildAndWritesPagesWithImages(t *testing.T) {
	root := writeBuild(t, twoComponents)
	fake := &fakeRenderer{}
	output, err := generate(context.Background(), root, generatedfile.Spec{Type: "storybook-docs", Source: "storybook-static", Output: "docs/stories", Title: "Our UI"}, startFake(fake))
	if err != nil {
		t.Fatal(err)
	}
	if fake.rendered.Load() != 3 {
		t.Fatalf("rendered %d stories, want 3", fake.rendered.Load())
	}
	index := fileBody(t, output.Files, "README.md")
	if !strings.Contains(index, "# Our UI") {
		t.Fatalf("index = %s", index)
	}
	page := fileBody(t, output.Files, "components/button/README.md")
	if !strings.Contains(page, "![Ghost](./images/ghost.png)") || !strings.Contains(page, "![Primary](./images/primary.png)") {
		t.Fatalf("page = %s", page)
	}
	if !strings.HasPrefix(fileBody(t, output.Files, "components/button/images/primary.png"), "png:c-button--primary@http") {
		t.Fatal("the image is not the render of the story over the served build")
	}
}

func TestGenerateFiltersStoriesByTitleWithTheInputPatterns(t *testing.T) {
	root := writeBuild(t, twoComponents)
	fake := &fakeRenderer{}
	output, err := generate(context.Background(), root, generatedfile.Spec{Source: "storybook-static", Input: []string{"!Internal/"}}, startFake(fake))
	if err != nil {
		t.Fatal(err)
	}
	if fake.rendered.Load() != 2 {
		t.Fatalf("rendered %d, want 2", fake.rendered.Load())
	}
	for _, name := range paths(output.Files) {
		if strings.HasPrefix(name, "internal/") {
			t.Fatalf("an excluded title was written: %s", name)
		}
	}
}

func TestGenerateReportsFailedStoriesAsWarningsAndKeepsTheRest(t *testing.T) {
	root := writeBuild(t, twoComponents)
	fake := &fakeRenderer{fail: map[string]bool{"c-button--ghost": true}}
	output, err := generate(context.Background(), root, generatedfile.Spec{Source: "storybook-static"}, startFake(fake))
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Warnings) != 1 || !strings.Contains(output.Warnings[0], "Ghost") {
		t.Fatalf("warnings = %v", output.Warnings)
	}
	if !strings.Contains(fileBody(t, output.Files, "components/button/README.md"), "![Primary]") {
		t.Fatal("the stories that rendered are missing")
	}
}

func TestGenerateExplainsWhatIsMissing(t *testing.T) {
	root := writeBuild(t, twoComponents)
	cases := map[string]generatedfile.Spec{
		"needs a source":    {},
		"was not found":     {Source: "nothing-here"},
		"has no index.json": {Source: "."},
	}
	for want, spec := range cases {
		_, err := generate(context.Background(), root, spec, startFake(&fakeRenderer{}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("spec %+v: err = %v, want it to contain %q", spec, err, want)
		}
	}
}
