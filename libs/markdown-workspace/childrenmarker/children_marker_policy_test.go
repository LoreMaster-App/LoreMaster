package childrenmarker

import "testing"

func TestParseRecognisesTheMarkerAndItsOptions(t *testing.T) {
	cases := map[string]Options{
		"[children]":                 {},
		"  [children]  ":             {},
		"[Children]":                 {},
		"[children all]":             {All: true},
		"[children depth=2]":         {Depth: 2},
		"[children  all  depth = 3]": {All: true, Depth: 3},
	}
	for text, want := range cases {
		got, ok := Parse(text)
		if !ok || got != want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", text, got, ok, want)
		}
	}
}

func TestParseRefusesAnythingElse(t *testing.T) {
	for _, text := range []string{"", "children", "[children] more", "see [children]", "[child]", "[children depth=x]", "[children, all]", "[children]()"} {
		if options, ok := Parse(text); ok {
			t.Errorf("Parse(%q) accepted: %+v", text, options)
		}
	}
}

func TestStringWritesWhatParseReads(t *testing.T) {
	for _, options := range []Options{{}, {All: true}, {Depth: 2}, {All: true, Depth: 4}} {
		back, ok := Parse(options.String())
		if !ok || back != options {
			t.Errorf("%+v -> %q -> %+v, %v", options, options.String(), back, ok)
		}
	}
}

func TestLevels(t *testing.T) {
	if (Options{}).Levels() != 1 || (Options{Depth: 3}).Levels() != 3 || (Options{All: true}).Levels() != 0 || (Options{All: true, Depth: 2}).Levels() != 0 {
		t.Fatal("levels are wrong")
	}
}
