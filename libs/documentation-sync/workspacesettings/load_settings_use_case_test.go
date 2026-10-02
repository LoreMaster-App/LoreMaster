package workspacesettings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func workspaceWith(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(root, FileName), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestLoadingAMissingFileGivesDefaultsAndAFirstSync(t *testing.T) {
	loaded, err := LoadSettings(workspaceWith(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{Version: 1, Outputs: []Output{defaultOutput()}}
	if !loaded.FirstSync || loaded.Exists || !reflect.DeepEqual(loaded.Settings, want) {
		t.Fatalf("loaded %+v", loaded)
	}
}

func TestLoadingFillsDefaultsAndTrimsThePrefix(t *testing.T) {
	loaded, err := LoadSettings(workspaceWith(t, `version: 1
outputs:
  - baseUrl: https://acme.atlassian.net/wiki
    space: ENG
    parentPageId: "123000"
    titlePrefix: "  MNCI "
    content:
      - roots: [docs]
        excludes: ["drafts/**"]
`))
	if err != nil {
		t.Fatal(err)
	}
	want := Output{
		Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG", ParentPageID: "123000", TitlePrefix: "MNCI",
		Direction: "to-platform", MermaidMode: "image", TitleCollision: "fail", LinkMode: "title",
		Content: []Content{{Type: "markdown", Roots: []string{"docs"}, Excludes: []string{"drafts/**"}, Template: "default"}},
	}
	if loaded.FirstSync || !loaded.Exists || !reflect.DeepEqual(loaded.Settings.Outputs[0], want) {
		t.Fatalf("\n got: %+v\nwant: %+v\nfirst sync %v", loaded.Settings.Outputs[0], want, loaded.FirstSync)
	}
}

func TestLoadingRefusesWhatItCannotTrust(t *testing.T) {
	cases := map[string]struct{ content, want string }{
		"misspelt key": {"version: 1\noutputs:\n  - platfrom: confluence\n", "line 3: field platfrom not found"},
		"secret":       {"version: 1\noutputs:\n  - space: ENG\n    apiToken: abc\n", `.lore-master.yaml line 4: "apiToken" looks like a secret; secrets never go in this committed file`},
		"not yaml":     {"version: [1\n", ".lore-master.yaml: yaml:"},
		"invalid":      {"version: 1\noutputs:\n  - direction: two-way\n", `outputs[0].direction "two-way" is planned but not available yet (#95); use to-platform`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := LoadSettings(workspaceWith(t, tc.content))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want it to contain %q", err, tc.want)
			}
		})
	}
}
