package siterender

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEveryBuiltinShellHasContentAndOnlyKnownPlaceholders(t *testing.T) {
	for _, name := range []string{"default", "minimal", "handbook"} {
		chosen, ok := BuiltinTemplate(name)
		if !ok {
			t.Fatalf("%s is not built in", name)
		}
		if problems := ShellProblems(chosen.Shell); len(problems) != 0 {
			t.Errorf("%s: %v", name, problems)
		}
	}
	if _, ok := BuiltinTemplate("fancy"); ok {
		t.Fatal("an unknown name was accepted")
	}
}

func TestShellProblemsNamesWhatIsWrong(t *testing.T) {
	problems := ShellProblems("<main>{{ contnet }}{{contnet}}</main>")

	if len(problems) != 2 || !strings.Contains(problems[0], "{{content}}") || !strings.Contains(problems[1], "{{contnet}}") {
		t.Fatalf("problems: %v", problems)
	}
	if len(ShellProblems("{{ content }}")) != 0 {
		t.Fatal("spaces inside the braces were refused")
	}
}

func TestACustomShellIsFilledPerPage(t *testing.T) {
	chosen := Template{
		Name:   "mine",
		Shell:  `<h1>{{title}}</h1><a href="{{root}}home">{{site_title}}</a>{{breadcrumbs}}{{nav}}<i>{{source}}</i>{{content}}`,
		Assets: []SiteFile{{Path: "assets/template/site.css", Content: []byte("body{}")}},
	}

	files, err := GenerateSiteTemplated("My <Project>", sampleTree(), nil, chosen)
	if err != nil {
		t.Fatal(err)
	}

	guide, _ := fileByPath(files, "readme.guide.html")
	for _, want := range []string{"<h1>Guide</h1>", `href="home"`, "My &lt;Project&gt;", `<a href="readme.html">Home</a>`, "<i>readme.guide.md</i>", `id="lm-markdown"`} {
		if !strings.Contains(string(guide.Content), want) {
			t.Errorf("readme.guide.html lacks %q:\n%s", want, guide.Content)
		}
	}
	setup, _ := fileByPath(files, "docs/setup.html")
	if !strings.Contains(string(setup.Content), `href="../home"`) {
		t.Errorf("a nested page should climb to the root:\n%s", setup.Content)
	}
	if _, ok := fileByPath(files, "assets/template/site.css"); !ok {
		t.Error("the template's own assets were not shipped")
	}
}

func TestBuiltinTemplatesChangeTheLookNotTheContent(t *testing.T) {
	var shells []string
	for _, name := range []string{"default", "minimal", "handbook"} {
		chosen, _ := BuiltinTemplate(name)
		files, err := GenerateSiteTemplated("Site", sampleTree(), nil, chosen)
		if err != nil {
			t.Fatal(err)
		}
		page, _ := fileByPath(files, "readme.html")
		if !strings.Contains(string(page.Content), `data-markdown="`) {
			t.Errorf("%s dropped the Markdown", name)
		}
		css, _ := fileByPath(files, "assets/lore-master.css")
		if !strings.Contains(string(css.Content), ".markdown-body") {
			t.Errorf("%s has no content styles", name)
		}
		shells = append(shells, string(page.Content))
	}
	if shells[0] == shells[1] || shells[1] == shells[2] {
		t.Fatal("the built-ins render the same page")
	}
}

func TestHeadlessEmitsTheTreeAndTheMarkdown(t *testing.T) {
	chosen, _ := BuiltinTemplate("headless")

	files, err := GenerateSiteTemplated("Site", sampleTree(), nil, chosen)
	if err != nil {
		t.Fatal(err)
	}

	manifest, ok := fileByPath(files, "site.json")
	if !ok {
		t.Fatalf("no site.json in %v", paths(files))
	}
	var got struct {
		Title string `json:"title"`
		Home  string `json:"home"`
		Pages []struct {
			Path     string   `json:"path"`
			Parent   string   `json:"parent"`
			Children []string `json:"children"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(manifest.Content, &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "Site" || got.Home != "readme.md" || len(got.Pages) != 3 {
		t.Fatalf("manifest: %+v", got)
	}
	if got.Pages[1].Path != "readme.guide.md" || got.Pages[1].Parent != "readme.md" || got.Pages[0].Children[0] != "readme.guide.md" {
		t.Fatalf("tree: %+v", got.Pages)
	}
	if markdown, ok := fileByPath(files, "docs/setup.md"); !ok || string(markdown.Content) != "# Setup\n" {
		t.Fatalf("the Markdown is not served at its own path: %v", paths(files))
	}
	if _, html := fileByPath(files, "readme.html"); html {
		t.Fatal("a headless site has HTML pages")
	}
}
