package workspacesettings

import (
	"strings"
	"testing"
)

func TestNamesMustBeDistinctAndPlain(t *testing.T) {
	cases := []struct {
		name  string
		names []string
		want  string
	}{
		{"same name twice", []string{"Technical docs", "technical DOCS"}, `outputs[1].name "technical DOCS" is already the name of outputs[0]; names must be different`},
		{"blank", []string{"   "}, "outputs[0].name is blank"},
		{"padded", []string{" docs"}, `outputs[0].name " docs" starts or ends with a space`},
		{"too long", []string{strings.Repeat("a", 61)}, "outputs[0].name is longer than 60 characters"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settings := valid()
			settings.Outputs = nil
			for i, name := range tc.names {
				output := valid().Outputs[0]
				output.Space = "S" + string(rune('A'+i))
				output.Name = name
				settings.Outputs = append(settings.Outputs, output)
			}
			if err := Validate(settings); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestDistinctNamesAreAccepted(t *testing.T) {
	settings := valid()
	second := settings.Outputs[0]
	second.Space, second.Name = "OTHER", "User manuals"
	settings.Outputs[0].Name = "Technical docs"
	settings.Outputs = append(settings.Outputs, second)

	if err := Validate(settings); err != nil {
		t.Fatal(err)
	}
}

func TestDisplayNameUsesTheNameOrMakesOneUp(t *testing.T) {
	outputs := []Output{
		{Platform: "confluence", Space: "ENG"},
		{Platform: "confluence", Space: "ENG"},
		{Platform: "confluence", Space: "ENG", Name: "Manuals"},
		{Platform: "github-pages", Repo: "acme/site"},
		{Platform: "github-wiki"},
	}
	want := []string{"Confluence · ENG", "Confluence · ENG (2)", "Manuals", "GitHub Pages · acme/site", "GitHub Wiki · this repository"}

	for i, expected := range want {
		if got := DisplayName(outputs, i); got != expected {
			t.Errorf("outputs[%d] = %q, want %q", i, got, expected)
		}
	}
}
