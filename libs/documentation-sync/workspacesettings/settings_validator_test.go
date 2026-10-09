package workspacesettings

import (
	"strings"
	"testing"
)

func valid() Settings {
	output := defaultOutput()
	output.BaseURL, output.Space, output.ParentPageID, output.TitlePrefix = "https://acme.atlassian.net/wiki", "ENG", "1", "MNCI"

	return Settings{Version: 1, Outputs: []Output{output}}
}

func TestValidate(t *testing.T) {
	if err := Validate(valid()); err != nil {
		t.Fatalf("valid settings refused: %v", err)
	}
	twoWay := valid()
	twoWay.Outputs[0].Direction = "two-way"
	if err := Validate(twoWay); err != nil {
		t.Fatalf("two-way direction refused: %v", err)
	}
	blankTemplate := valid()
	blankTemplate.Outputs[0].Content[0].Template = ""
	if err := Validate(blankTemplate); err != nil {
		t.Fatalf("an empty template (meaning default) was refused: %v", err)
	}
	cases := []struct {
		name   string
		change func(*Settings)
		want   string
	}{
		{"future version", func(s *Settings) { s.Version = 2 }, "version 2 is not supported; this version of LoreMaster reads version 1"},
		{"no outputs", func(s *Settings) { s.Outputs = nil }, "outputs is empty; add at least one output"},
		{"unknown platform", func(s *Settings) { s.Outputs[0].Platform = "wiki" }, `outputs[0].platform "wiki" is not one of confluence, github-pages`},
		{"reserved platform", func(s *Settings) { s.Outputs[0].Platform = "notion" }, `outputs[0].platform "notion" is planned but not available yet (#166); use confluence or github-pages`},
		{"bad direction", func(s *Settings) { s.Outputs[0].Direction = "sideways" }, `outputs[0].direction "sideways" is not one of to-platform, two-way`},
		{"html macro reserved", func(s *Settings) { s.Outputs[0].MermaidMode = "html-macro" }, `outputs[0].mermaidMode "html-macro" is planned but not available yet (#40); use image or code`},
		{"test results reserved", func(s *Settings) { s.Outputs[0].Content[0].Type = "test-results" }, `outputs[0].content[0].type "test-results" is planned but not available yet (#96); use markdown`},
		{"custom template reserved", func(s *Settings) { s.Outputs[0].Content[0].Template = "fancy" }, `outputs[0].content[0].template "fancy": custom templates are planned but not available yet (#97); use default`},
		{"bad collision mode", func(s *Settings) { s.Outputs[0].TitleCollision = "overwrite" }, `outputs[0].titleCollision "overwrite" is not one of fail, adopt`},
		{"bad link mode", func(s *Settings) { s.Outputs[0].LinkMode = "url" }, `outputs[0].linkMode "url" is not one of title, id`},
		{"http address", func(s *Settings) { s.Outputs[0].BaseURL = "http://acme.com" }, `outputs[0].baseUrl "http://acme.com" must be an https address`},
		{"root outside", func(s *Settings) { s.Outputs[0].Content[0].Roots = []string{"../other"} }, `outputs[0].content[0].roots entry "../other" must be a path inside the workspace`},
		{"no content", func(s *Settings) { s.Outputs[0].Content = nil }, "outputs[0].content is empty; add at least one content entry"},
		{"absolute path", func(s *Settings) { s.Outputs[0].Path = "/docs" }, `outputs[0].path "/docs" must be relative to the branch root, without a leading slash`},
		{"path climbs out", func(s *Settings) { s.Outputs[0].Path = "../docs" }, `outputs[0].path "../docs" must be a plain folder path inside the branch`},
		{"path with a backslash", func(s *Settings) { s.Outputs[0].Path = `docs\site` }, "must not contain spaces or backslashes"},
		{"same destination twice with the same path", func(s *Settings) {
			s.Outputs[0].Path = "docs"
			second := validPagesOutput()
			second.Path = "docs/"
			s.Outputs = append(s.Outputs, second)
		}, "outputs[1] publishes to the same repo and branch as outputs[0]; two outputs would overwrite each other"},
		{"same destination twice", func(s *Settings) { s.Outputs = append(s.Outputs, s.Outputs[0]) }, "outputs[1] syncs to the same space and parent page as outputs[0]; two outputs would overwrite each other's pages"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settings := valid()
			tc.change(&settings)
			if err := Validate(settings); err == nil || err.Error() != tc.want {
				t.Fatalf("\n got: %v\nwant: %s", err, tc.want)
			}
		})
	}
}

func validPagesOutput() Output {
	return Output{
		Platform:  "github-pages",
		Direction: "to-platform",
		Content:   []Content{{Type: "markdown", Roots: []string{"."}, Template: "default"}},
	}
}

func TestValidateGitHubPages(t *testing.T) {
	valid := func() Settings { return Settings{Version: 1, Outputs: []Output{validPagesOutput()}} }

	if err := Validate(valid()); err != nil {
		t.Fatalf("a minimal github-pages output (own repo, gh-pages) was refused: %v", err)
	}
	explicit := valid()
	explicit.Outputs[0].Repo, explicit.Outputs[0].Branch = "russoedu/LoreMaster", "docs"
	if err := Validate(explicit); err != nil {
		t.Fatalf("an explicit repo and branch were refused: %v", err)
	}

	cases := []struct {
		name   string
		change func(*Settings)
		want   string
	}{
		{"two-way refused", func(s *Settings) { s.Outputs[0].Direction = "two-way" }, `outputs[0].direction "two-way" is not one of to-platform`},
		{"confluence field refused", func(s *Settings) { s.Outputs[0].Space = "ENG" }, "outputs[0].space is not used by a github-pages output; remove it"},
		{"base url refused", func(s *Settings) { s.Outputs[0].BaseURL = "https://x.atlassian.net/wiki" }, "outputs[0].baseUrl is not used by a github-pages output; remove it"},
		{"bad branch", func(s *Settings) { s.Outputs[0].Branch = "feature branch" }, `outputs[0].branch "feature branch" is not a valid branch name`},
		{"repo with spaces", func(s *Settings) { s.Outputs[0].Repo = "owner name" }, `outputs[0].repo "owner name" must not contain spaces`},
		{"same destination twice", func(s *Settings) { s.Outputs = append(s.Outputs, validPagesOutput()) }, "outputs[1] publishes to the same repo and branch as outputs[0]; two outputs would overwrite each other"},
		{"content still checked", func(s *Settings) { s.Outputs[0].Content[0].Template = "fancy" }, `outputs[0].content[0].template "fancy": custom templates are planned but not available yet (#97); use default`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settings := valid()
			tc.change(&settings)
			if err := Validate(settings); err == nil || err.Error() != tc.want {
				t.Fatalf("\n got: %v\nwant: %s", err, tc.want)
			}
		})
	}
}

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	settings := valid()
	settings.Version = 9
	settings.Outputs[0].LinkMode = "x"
	want := "version 9 is not supported; this version of LoreMaster reads version 1\n" + `outputs[0].linkMode "x" is not one of title, id`
	if err := Validate(settings); err == nil || err.Error() != want {
		t.Fatalf("\n got: %v\nwant: %s", err, want)
	}
}

func settingsWith(generators ...Generator) Settings {
	return Settings{Version: CurrentVersion, Outputs: []Output{defaultOutput()}, Generators: generators}
}

func TestValidateAcceptsABuiltGenerator(t *testing.T) {
	err := Validate(settingsWith(Generator{Type: "test-results", Input: []string{"**/junit*.xml", "!vendor/"}, Output: "docs/tests", Title: "Results"}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateRefusesAGeneratorThatCannotBeUsed(t *testing.T) {
	cases := []struct {
		name      string
		generator Generator
		want      string
	}{
		{"unknown type", Generator{Type: "crayon", Output: "docs/x"}, `generators[0].type "crayon" is not one of test-results, go-docs, openapi-docs, python-docs, csharp-docs, dart-docs, ts-docs`},
		{"no output", Generator{Type: "test-results"}, "generators[0].output is empty"},
		{"output is the workspace", Generator{Type: "test-results", Output: "."}, "must be a folder inside the workspace"},
		{"output escapes", Generator{Type: "test-results", Output: "../out"}, "must be a folder inside the workspace"},
		{"absolute output", Generator{Type: "test-results", Output: "/out"}, "must be a folder inside the workspace"},
		{"empty input pattern", Generator{Type: "test-results", Output: "docs/x", Input: []string{" "}}, "generators[0].input[0]"},
		{"input escapes", Generator{Type: "test-results", Output: "docs/x", Input: []string{"../reports"}}, "generators[0].input[0]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(settingsWith(tc.generator))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestValidateRefusesGeneratorsThatWriteIntoTheSameFolder(t *testing.T) {
	for _, outputs := range [][2]string{{"docs/tests", "docs/tests"}, {"docs", "docs/tests"}, {"docs/tests/", "docs"}} {
		err := Validate(settingsWith(Generator{Type: "test-results", Output: outputs[0]}, Generator{Type: "test-results", Output: outputs[1]}))
		if err == nil || !strings.Contains(err.Error(), "overlapping folders") {
			t.Fatalf("%v: got %v", outputs, err)
		}
	}
	if err := Validate(settingsWith(Generator{Type: "test-results", Output: "docs/tests"}, Generator{Type: "test-results", Output: "docs/testing"})); err != nil {
		t.Fatalf("sibling folders with a shared prefix are fine: %v", err)
	}
}
