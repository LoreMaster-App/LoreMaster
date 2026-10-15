package templatereference

import "testing"

func TestParseReadsEveryForm(t *testing.T) {
	cases := map[string]Reference{
		"":                      {Kind: Builtin, Name: "default"},
		"default":               {Kind: Builtin, Name: "default"},
		"handbook":              {Kind: Builtin, Name: "handbook"},
		"12345":                 {Kind: Page, PageID: "12345"},
		"page: 77":              {Kind: Page, PageID: "77"},
		"https://x.atlassian.net/wiki/spaces/DOC/pages/987/Template": {Kind: Page, PageID: "987"},
		"https://wiki.example/pages/viewpage.action?pageId=55":       {Kind: Page, PageID: "55"},
		"templates/site":        {Kind: File, Path: "templates/site"},
		`templates\wiki.md`:     {Kind: File, Path: "templates/wiki.md"},
		"layout.xhtml":          {Kind: File, Path: "layout.xhtml"},
		"./brand":               {Kind: File, Path: "brand"},
	}
	for value, want := range cases {
		got, ok := Parse(value)
		if !ok || got != want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", value, got, ok, want)
		}
	}
}

func TestParseRefusesWhatLeavesTheWorkspace(t *testing.T) {
	for _, value := range []string{"../out", "/etc/x", "C:/x/y", ".", "https://example.com/nothing"} {
		if reference, ok := Parse(value); ok {
			t.Errorf("Parse(%q) accepted %+v", value, reference)
		}
	}
}

func TestCheckHoldsEachPlatformToItsOwnTemplates(t *testing.T) {
	good := map[string][]string{
		"github-pages": {"", "minimal", "headless", "templates/site"},
		"github-wiki":  {"default", "templates/wiki.md"},
		"confluence":   {"", "123", "layout.xhtml"},
	}
	for platform, values := range good {
		for _, value := range values {
			if _, problem := Check(platform, value); problem != "" {
				t.Errorf("%s %q refused: %s", platform, value, problem)
			}
		}
	}
	bad := map[string][]string{
		"github-pages": {"123", "fancy"},
		"github-wiki":  {"handbook", "templates/site"},
		"confluence":   {"minimal", "../x"},
	}
	for platform, values := range bad {
		for _, value := range values {
			if _, problem := Check(platform, value); problem == "" {
				t.Errorf("%s %q was accepted", platform, value)
			}
		}
	}
}
