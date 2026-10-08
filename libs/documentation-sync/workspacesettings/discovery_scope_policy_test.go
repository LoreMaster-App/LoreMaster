package workspacesettings

import (
	"slices"
	"strings"
	"testing"
)

func TestDiscoveryScopeSkipsGitignoredByDefault(t *testing.T) {
	off := false
	on := true
	for _, tc := range []struct {
		name string
		flag *bool
		want bool
	}{{"absent", nil, true}, {"true", &on, true}, {"false", &off, false}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := (Settings{SkipGitignored: tc.flag}).DiscoveryScope().SkipGitignored; got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExcludesForPutsTheIgnoreListBeforeTheContentExcludes(t *testing.T) {
	scope := Settings{Ignore: []string{"drafts/**", "NOTES.md"}}.DiscoveryScope()
	got := scope.ExcludesFor(Content{Excludes: []string{"internal/"}})
	if want := []string{"drafts/**", "NOTES.md", "internal/"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := (DiscoveryScope{}).ExcludesFor(Content{}); len(got) != 0 {
		t.Fatalf("an empty scope and entry excluded %q", got)
	}
}

func TestValidateRefusesIgnorePatternsOutsideTheWorkspace(t *testing.T) {
	settings := Settings{Version: CurrentVersion, Outputs: []Output{defaultOutput()}, Ignore: []string{"drafts/**", "", "../other", "/etc"}}
	err := Validate(settings)
	if err == nil {
		t.Fatal("expected a problem")
	}
	for _, at := range []string{`ignore[1]`, `ignore[2]`, `ignore[3]`} {
		if !strings.Contains(err.Error(), at) {
			t.Fatalf("%s not reported in %v", at, err)
		}
	}
	if strings.Contains(err.Error(), "ignore[0]") {
		t.Fatalf("a valid pattern was reported: %v", err)
	}
}
