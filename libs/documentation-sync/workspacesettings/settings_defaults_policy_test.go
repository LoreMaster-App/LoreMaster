package workspacesettings

import (
	"reflect"
	"testing"
)

func TestAWorkspaceNotSetUpReadsAsTheDefaultsAndAFirstSync(t *testing.T) {
	loaded := NotSetUp()

	want := Settings{Version: 1, Outputs: []Output{defaultOutput()}}
	if !loaded.FirstSync || loaded.Exists || !reflect.DeepEqual(loaded.Settings, want) {
		t.Fatalf("loaded %+v", loaded)
	}
}

func TestResolvingFillsDefaultsAndTrimsThePrefix(t *testing.T) {
	resolved, firstSync, err := ResolveSettings(Settings{Outputs: []Output{{
		BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG", ParentPageID: "123000", TitlePrefix: "  MNCI ",
		Content: []Content{{Roots: []string{"docs"}, Excludes: []string{"drafts/**"}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}

	want := Output{
		Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG", ParentPageID: "123000", TitlePrefix: "MNCI",
		Direction: "to-platform", MermaidMode: "image", MermaidMaxWidth: "800px", TitleCollision: "fail", LinkMode: "title",
		Content: []Content{{Type: "markdown", Roots: []string{"docs"}, Excludes: []string{"drafts/**"}, Template: "default"}},
	}
	if firstSync || !reflect.DeepEqual(resolved.Outputs[0], want) {
		t.Fatalf("\n got: %+v\nwant: %+v\nfirst sync %v", resolved.Outputs[0], want, firstSync)
	}
}

func TestResolvingRefusesWhatItCannotTrust(t *testing.T) {
	_, _, err := ResolveSettings(Settings{Outputs: []Output{{Direction: "sideways"}}})

	if err == nil || err.Error() == "" {
		t.Fatal("a direction that does not exist must be refused")
	}
}

func TestTheDiscoveryScopeFollowsTheSettings(t *testing.T) {
	no := false
	scope := Settings{SkipGitignored: &no, Ignore: []string{"drafts/**", "NOTES.md"}}.DiscoveryScope()

	if scope.SkipGitignored || !reflect.DeepEqual(scope.Ignore, []string{"drafts/**", "NOTES.md"}) {
		t.Fatalf("scope %+v", scope)
	}
}
