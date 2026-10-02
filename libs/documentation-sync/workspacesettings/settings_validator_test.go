package workspacesettings

import (
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
	cases := []struct {
		name   string
		change func(*Settings)
		want   string
	}{
		{"future version", func(s *Settings) { s.Version = 2 }, "version 2 is not supported; this version of Lore Master reads version 1"},
		{"no outputs", func(s *Settings) { s.Outputs = nil }, "outputs is empty; add at least one output"},
		{"unknown platform", func(s *Settings) { s.Outputs[0].Platform = "notion" }, `outputs[0].platform "notion" is not one of confluence`},
		{"two-way reserved", func(s *Settings) { s.Outputs[0].Direction = "two-way" }, `outputs[0].direction "two-way" is planned but not available yet (#95); use to-platform`},
		{"html macro reserved", func(s *Settings) { s.Outputs[0].MermaidMode = "html-macro" }, `outputs[0].mermaidMode "html-macro" is planned but not available yet (#40); use image or code`},
		{"test results reserved", func(s *Settings) { s.Outputs[0].Content[0].Type = "test-results" }, `outputs[0].content[0].type "test-results" is planned but not available yet (#96); use markdown`},
		{"custom template reserved", func(s *Settings) { s.Outputs[0].Content[0].Template = "fancy" }, `outputs[0].content[0].template "fancy": custom templates are planned but not available yet (#97); use default`},
		{"bad collision mode", func(s *Settings) { s.Outputs[0].TitleCollision = "overwrite" }, `outputs[0].titleCollision "overwrite" is not one of fail, adopt`},
		{"bad link mode", func(s *Settings) { s.Outputs[0].LinkMode = "url" }, `outputs[0].linkMode "url" is not one of title, id`},
		{"http address", func(s *Settings) { s.Outputs[0].BaseURL = "http://acme.com" }, `outputs[0].baseUrl "http://acme.com" must be an https address`},
		{"root outside", func(s *Settings) { s.Outputs[0].Content[0].Roots = []string{"../other"} }, `outputs[0].content[0].roots entry "../other" must be a path inside the workspace`},
		{"no content", func(s *Settings) { s.Outputs[0].Content = nil }, "outputs[0].content is empty; add at least one content entry"},
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

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	settings := valid()
	settings.Version = 9
	settings.Outputs[0].LinkMode = "x"
	want := "version 9 is not supported; this version of Lore Master reads version 1\n" + `outputs[0].linkMode "x" is not one of title, id`
	if err := Validate(settings); err == nil || err.Error() != want {
		t.Fatalf("\n got: %v\nwant: %s", err, want)
	}
}
